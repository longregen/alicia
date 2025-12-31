# Building secure MCP clients with microVM isolation

Running untrusted MCP servers safely requires **hardware-level isolation** that containers cannot provide. This primer presents a complete architecture combining Go orchestration, microvm.nix for declarative NixOS virtual machines, and the Firecracker VMM to create security boundaries between MCP servers and the host system. The approach draws from production patterns at AWS Lambda and Fly.io, adapted for the MCP ecosystem.

The key insight is that MCP servers, which can execute arbitrary tools and access external resources, represent a significant attack surface. By running each server in its own microVM with restricted networking and filesystem access, we achieve **defense-in-depth**: even if an MCP server is compromised, it cannot access other servers' data or the host system.

---

## Part 1: MCP client implementation fundamentals

### Protocol architecture

MCP uses **JSON-RPC 2.0** over two transport types: stdio for local servers (spawned as subprocesses) and Streamable HTTP for remote servers. The protocol follows a client-host-server model where the host application (like Claude Desktop) manages multiple client connections to different servers.

Every message must be UTF-8 encoded. Requests expect responses, while notifications are fire-and-forget:

```json
// Request
{"jsonrpc": "2.0", "id": 1, "method": "tools/call", 
 "params": {"name": "get_weather", "arguments": {"city": "London"}}}

// Response  
{"jsonrpc": "2.0", "id": 1, 
 "result": {"content": [{"type": "text", "text": "22°C"}]}}

// Notification (no id, no response expected)
{"jsonrpc": "2.0", "method": "notifications/progress", 
 "params": {"progress": 50, "total": 100}}
```

### The initialization handshake

Before any tool calls, clients must complete a three-step handshake that negotiates protocol version and capabilities:

```
Client                              Server
  │                                   │
  │──── initialize request ──────────▶│
  │                                   │
  │◀──── capabilities response ───────│
  │                                   │
  │──── initialized notification ────▶│
  │                                   │
  └───── operational phase ───────────┘
```

The initialize request declares the client's capabilities (sampling, roots), and the server responds with its capabilities (tools, resources, prompts, logging). Only after the client sends the `initialized` notification can regular operations begin.

### Building a Go MCP client

The official Go SDK provides a clean interface for client implementation:

```go
import (
    "context"
    "os/exec"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

func connectToServer(ctx context.Context, command string, args []string) (*mcp.Session, error) {
    client := mcp.NewClient(&mcp.Implementation{
        Name:    "my-orchestrator",
        Version: "1.0.0",
    }, nil)
    
    transport := &mcp.CommandTransport{
        Command: exec.Command(command, args...),
    }
    
    session, err := client.Connect(ctx, transport, nil)
    if err != nil {
        return nil, err
    }
    
    // Session is now initialized and ready for tool calls
    return session, nil
}

// Call a tool
func callTool(ctx context.Context, session *mcp.Session, name string, args map[string]any) error {
    result, err := session.CallTool(ctx, &mcp.CallToolParams{
        Name:      name,
        Arguments: args,
    })
    if err != nil {
        return err
    }
    // Process result.Content
    return nil
}
```

For stdio transport, the client spawns the server as a subprocess and communicates via newline-delimited JSON messages on stdin/stdout. The server's stderr can be used for logging without interfering with the protocol.

### Streamable HTTP transport

Remote servers use HTTP POST for requests, with the server responding either with `application/json` or `text/event-stream` (SSE) for streaming responses. Session management uses the `Mcp-Session-Id` header:

```go
// HTTP request headers
Accept: application/json, text/event-stream
MCP-Protocol-Version: 2025-06-18
Mcp-Session-Id: <session-id>  // After initialization
```

Security critical: Validate the `Origin` header to prevent DNS rebinding attacks, and bind to localhost only for local development servers.

---

## Part 2: MicroVM isolation with microvm.nix

### Why microVMs over containers

Containers share the host kernel, meaning a kernel vulnerability can lead to container escape. MicroVMs run their own kernel, reducing the attack surface to the hypervisor's device emulation code. Firecracker, written in **~50,000 lines of Rust** (versus QEMU's 1.4 million lines of C), provides a minimal attack surface while maintaining sub-200ms boot times.

microvm.nix provides a declarative, NixOS-native way to define and manage these VMs with reproducible builds.

### Hypervisor selection guide

| Hypervisor | Language | virtiofs | vsock | Security | Boot time | Best for |
|------------|----------|----------|-------|----------|-----------|----------|
| **Firecracker** | Rust | ❌ | ✅ | Excellent | ~125ms | Untrusted code |
| **cloud-hypervisor** | Rust | ✅ | ✅ | Very good | ~200ms | Needs file sharing |
| **QEMU** | C | ✅ | ✅ | Good | ~500ms | Development |

For MCP server isolation where servers need no shared filesystem, **Firecracker** is the clear choice. If servers need access to shared directories (like a codebase), **cloud-hypervisor** with virtiofs provides the best security/functionality tradeoff.

### Basic microvm.nix configuration

```nix
# flake.nix
{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    microvm.url = "github:astro/microvm.nix";
    microvm.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = { self, nixpkgs, microvm }: {
    nixosConfigurations.mcp-sandbox = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      modules = [
        microvm.nixosModules.microvm
        ./mcp-server-vm.nix
      ];
    };
  };
}
```

```nix
# mcp-server-vm.nix
{ pkgs, ... }: {
  networking.hostName = "mcp-sandbox";
  
  microvm = {
    hypervisor = "cloud-hypervisor";  # Or "firecracker"
    mem = 512;
    vcpu = 2;
    
    # Network: vsock only for controlled communication
    interfaces = [];  # No network interfaces
    vsock.cid = 100;  # Context ID for this VM
    
    # Filesystem: read-only root, persistent data volume
    storeDiskType = "squashfs";  # Compact, read-only
    storeOnDisk = true;          # Don't mount host /nix/store
    
    volumes = [{
      image = "data.img";
      mountPoint = "/data";
      size = 256;  # MiB
      autoCreate = true;
    }];
  };
  
  # Minimal services
  services.openssh.enable = false;
  
  # Install MCP server
  environment.systemPackages = with pkgs; [
    nodejs_20
    # Your MCP server package
  ];
  
  system.stateVersion = "24.05";
}
```

### Network isolation patterns

**Complete isolation with vsock** (recommended for untrusted servers):
```nix
microvm = {
  interfaces = [];     # No network
  vsock.cid = 100;     # vsock only
};
```

**TAP interface with host bridge** (for servers needing network access):
```nix
# In the VM configuration
microvm.interfaces = [{
  type = "tap";
  id = "vm-mcp-01";
  mac = "02:00:00:00:00:01";
}];

# On the host
systemd.network.netdevs."10-microvm".netdevConfig = {
  Kind = "bridge";
  Name = "microvm";
};

systemd.network.networks."11-microvm" = {
  matchConfig.Name = "vm-*";
  networkConfig.Bridge = "microvm";
};
```

### Filesystem security

The default read-only root filesystem provides strong integrity guarantees. For servers needing write access:

```nix
# Writable overlay for /nix/store (if server needs to install packages)
microvm = {
  writableStoreOverlay = "/nix/.rw-store";
  
  shares = [{
    proto = "virtiofs";
    tag = "rw-store";
    source = "/var/lib/microvms/my-server/rw-store";
    mountPoint = "/nix/.rw-store";
  }];
};
```

For cloud-hypervisor/QEMU, **virtiofs** provides faster shared directory access than 9p. Firecracker doesn't support filesystem shares—use volumes or include everything in the root image.

---

## Part 3: MCP server isolation patterns

### Container baseline (before migrating to VMs)

Many teams start with Docker isolation before graduating to VMs:

```yaml
# docker-compose.yml - hardened MCP server
services:
  mcp-server:
    image: my-mcp-server:latest
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    read_only: true
    tmpfs:
      - /tmp
    networks:
      - mcp-internal
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
          pids: 100

networks:
  mcp-internal:
    internal: true  # No external access
```

This provides namespace isolation but shares the host kernel. For stronger isolation, use **gVisor** (`--runtime runsc`) or migrate to VMs.

### Credential handling architecture

Never store credentials in plaintext configuration files. Use a layered approach:

**Level 1 - Environment injection** (development only):
```json
{
  "mcpServers": {
    "github": {
      "command": "node",
      "args": ["server.js"],
      "env": {"GITHUB_TOKEN": "${GITHUB_TOKEN}"}
    }
  }
}
```

**Level 2 - Secrets manager integration** (recommended):
```bash
# Using 1Password CLI
op run --env-file=mcp.env -- node mcp-server.js

# mcp.env references vault paths
GITHUB_TOKEN=op://vault/github/token
```

**Level 3 - Runtime injection via vsock** (for VM isolation):
```go
// Orchestrator retrieves secret and sends via vsock
secret := vault.GetSecret("mcp-server-1/github-token")
conn, _ := vsock.Dial(vmCID, secretPort, nil)
conn.Write([]byte(secret))
```

The VM's init process reads from vsock and sets environment variables without persisting to disk.

### Network security for MCP servers

Implement egress filtering to prevent data exfiltration:

```nix
# In VM configuration - firewall rules
networking.firewall = {
  enable = true;
  allowedTCPPorts = [];  # No inbound
  
  # Egress filtering via nftables
  extraCommands = ''
    # Allow only specific API endpoints
    iptables -A OUTPUT -d api.github.com -p tcp --dport 443 -j ACCEPT
    iptables -A OUTPUT -d api.openai.com -p tcp --dport 443 -j ACCEPT
    iptables -A OUTPUT -j DROP
  '';
};
```

For complete isolation, run with no network interfaces and proxy all requests through the orchestrator via vsock.

---

## Part 4: Go binary orchestration

### Firecracker Go SDK fundamentals

The firecracker-go-sdk provides the cleanest interface for VM management:

```go
import (
    firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
    models "github.com/firecracker-microvm/firecracker-go-sdk/client/models"
)

type MCPServerVM struct {
    machine    *firecracker.Machine
    socketPath string
    vsockCID   uint32
}

func (m *MCPServerVM) Create(ctx context.Context, rootfsPath string) error {
    cfg := firecracker.Config{
        SocketPath:      m.socketPath,
        KernelImagePath: "/var/lib/firecracker/vmlinux",
        KernelArgs:      "console=ttyS0 reboot=k panic=1 init=/init",
        Drives: []models.Drive{{
            DriveID:      firecracker.String("rootfs"),
            PathOnHost:   firecracker.String(rootfsPath),
            IsRootDevice: firecracker.Bool(true),
            IsReadOnly:   firecracker.Bool(true),
        }},
        MachineCfg: models.MachineConfiguration{
            VcpuCount:  firecracker.Int64(2),
            MemSizeMib: firecracker.Int64(512),
        },
        VsockDevices: []firecracker.VsockDevice{{
            Path: m.socketPath + ".vsock",
            CID:  m.vsockCID,
        }},
        // Jailer for additional isolation
        JailerCfg: &firecracker.JailerConfig{
            UID:           firecracker.Int(65534),  // nobody
            GID:           firecracker.Int(65534),
            ChrootBaseDir: "/srv/jailer",
            Daemonize:     false,
        },
    }
    
    cmd := firecracker.VMCommandBuilder{}.
        WithSocketPath(m.socketPath).
        WithBin("/usr/bin/firecracker").
        Build(ctx)
    
    machine, err := firecracker.NewMachine(ctx, cfg, firecracker.WithProcessRunner(cmd))
    if err != nil {
        return err
    }
    
    m.machine = machine
    return machine.Start(ctx)
}
```

### Vsock communication patterns

Vsock enables direct host-guest communication without network configuration:

```go
import "github.com/mdlayher/vsock"

// Host side: listen for connections from VMs
func (o *Orchestrator) listenVsock(port uint32) error {
    listener, err := vsock.Listen(port, nil)
    if err != nil {
        return err
    }
    
    for {
        conn, err := listener.Accept()
        if err != nil {
            continue
        }
        
        // Get VM's CID to identify which server connected
        addr := conn.RemoteAddr().(*vsock.Addr)
        vmCID := addr.ContextID
        
        go o.handleMCPConnection(conn, vmCID)
    }
}

// Guest side: connect to host
func connectToHost(port uint32) (net.Conn, error) {
    return vsock.Dial(vsock.Host, port, nil)  // CID 2 = host
}
```

For MCP protocol forwarding, run a proxy on the host that accepts vsock connections and translates to/from the client:

```go
func (o *Orchestrator) handleMCPConnection(vsockConn net.Conn, vmCID uint32) {
    // Find the MCP session for this VM
    session := o.sessions[vmCID]
    
    // Bidirectional copy - MCP messages flow through vsock
    go io.Copy(session.stdin, vsockConn)   // Client -> VM
    io.Copy(vsockConn, session.stdout)     // VM -> Client
}
```

### Lifecycle management

```go
type VMManager struct {
    vms       map[uint32]*MCPServerVM
    mu        sync.RWMutex
    healthTick *time.Ticker
}

func (m *VMManager) HealthCheck(ctx context.Context) {
    for range m.healthTick.C {
        m.mu.RLock()
        for cid, vm := range m.vms {
            go func(cid uint32, vm *MCPServerVM) {
                ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
                defer cancel()
                
                conn, err := vsock.DialContext(ctx, cid, healthPort, nil)
                if err != nil {
                    log.Printf("VM %d unhealthy: %v", cid, err)
                    m.RestartVM(cid)
                    return
                }
                conn.Close()
            }(cid, vm)
        }
        m.mu.RUnlock()
    }
}

func (m *VMManager) GracefulShutdown(ctx context.Context, cid uint32, timeout time.Duration) error {
    vm := m.vms[cid]
    
    // Send shutdown signal via vsock
    conn, _ := vsock.Dial(cid, shutdownPort, nil)
    conn.Write([]byte("shutdown"))
    conn.Close()
    
    // Wait with timeout
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    
    done := make(chan struct{})
    go func() {
        vm.machine.Wait(ctx)
        close(done)
    }()
    
    select {
    case <-done:
        return nil
    case <-ctx.Done():
        return vm.machine.StopVMM()  // Force kill
    }
}
```

### Dynamic Nix configuration generation

For programmatic VM creation, generate Nix configurations from Go:

```go
const vmTemplate = `
{ pkgs, ... }: {
  networking.hostName = "{{.Hostname}}";
  microvm = {
    hypervisor = "{{.Hypervisor}}";
    mem = {{.MemoryMB}};
    vcpu = {{.VCPUs}};
    vsock.cid = {{.VsockCID}};
    interfaces = [];
  };
  
  environment.systemPackages = [
    {{range .Packages}}pkgs.{{.}}
    {{end}}
  ];
  
  systemd.services.mcp-server = {
    wantedBy = ["multi-user.target"];
    serviceConfig = {
      ExecStart = "{{.MCPCommand}}";
      Environment = [
        {{range $k, $v := .Environment}}"{{$k}}={{$v}}"
        {{end}}
      ];
    };
  };
  
  system.stateVersion = "24.05";
}
`

func generateVMConfig(spec MCPServerSpec) (string, error) {
    tmpl := template.Must(template.New("vm").Parse(vmTemplate))
    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, spec); err != nil {
        return "", err
    }
    return buf.String(), nil
}

func buildVM(ctx context.Context, nixConfig string) (string, error) {
    // Write config to temp file
    configPath := filepath.Join(os.TempDir(), "vm-config.nix")
    os.WriteFile(configPath, []byte(nixConfig), 0644)
    
    // Build with nix
    cmd := exec.CommandContext(ctx, "nix", "build", 
        "--out-link", "/tmp/result",
        "-f", configPath)
    return "/tmp/result", cmd.Run()
}
```

---

## Part 5: Complete architecture

### System overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     Go Orchestrator (Host)                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ MCP Client   │  │ VM Manager   │  │ Config Gen   │          │
│  │ (Sessions)   │  │ (Lifecycle)  │  │ (Nix Build)  │          │
│  └──────┬───────┘  └──────┬───────┘  └──────────────┘          │
│         │                 │                                      │
│         │     vsock       │    Firecracker                      │
│         │   connections   │    processes                         │
├─────────┴─────────────────┴──────────────────────────────────────┤
│                     KVM / Host Kernel                            │
├──────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │  MicroVM 1  │  │  MicroVM 2  │  │  MicroVM 3  │              │
│  │ ┌─────────┐ │  │ ┌─────────┐ │  │ ┌─────────┐ │              │
│  │ │MCP Srv A│ │  │ │MCP Srv B│ │  │ │MCP Srv C│ │              │
│  │ └─────────┘ │  │ └─────────┘ │  │ └─────────┘ │              │
│  │  CID: 100   │  │  CID: 101   │  │  CID: 102   │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
└──────────────────────────────────────────────────────────────────┘
```

### Security boundaries and trust model

The architecture provides **three isolation layers**:

1. **KVM hardware virtualization**: Primary boundary. Guest cannot access host memory or other guests.
2. **Firecracker's minimal VMM**: Only 5 emulated devices, ~40 allowed syscalls via seccomp.
3. **Jailer sandbox**: Firecracker process runs in chroot with dropped privileges.

Trust levels:
- **Trusted**: Host kernel, Go orchestrator, Nix build system
- **Semi-trusted**: Firecracker VMM process (in jail)
- **Untrusted**: MCP server code inside VM

### Configuration flow

```
User Config (YAML)          Security Validation          Nix Generation
     │                            │                           │
     ▼                            ▼                           ▼
┌──────────┐              ┌──────────────┐            ┌──────────────┐
│ command  │──validate───▶│ Allowlisted  │──generate─▶│ microvm.nix  │
│ env vars │              │ commands only│            │ expression   │
│ resources│              │ No host paths│            │              │
└──────────┘              └──────────────┘            └──────────────┘
                                                            │
                                                     nix build
                                                            │
                                                            ▼
                               ┌──────────────────────────────────┐
                               │ VM rootfs (squashfs, read-only)  │
                               │ + runtime secrets via vsock      │
                               └──────────────────────────────────┘
```

### MCP request routing

```go
type Orchestrator struct {
    sessions   map[string]*mcp.Session    // External client sessions
    vms        map[uint32]*MCPServerVM    // VM by CID
    serverToVM map[string]uint32          // MCP server name -> VM CID
}

func (o *Orchestrator) RouteToolCall(ctx context.Context, serverName string, 
    tool string, args map[string]any) (*mcp.CallToolResult, error) {
    
    cid, ok := o.serverToVM[serverName]
    if !ok {
        return nil, fmt.Errorf("unknown server: %s", serverName)
    }
    
    // Connect to MCP server in VM via vsock
    conn, err := vsock.DialContext(ctx, cid, mcpPort, nil)
    if err != nil {
        return nil, fmt.Errorf("VM unreachable: %w", err)
    }
    defer conn.Close()
    
    // Forward the tool call
    request := mcp.CallToolRequest{
        Method: "tools/call",
        Params: mcp.CallToolParams{Name: tool, Arguments: args},
    }
    
    // JSON-RPC over vsock
    encoder := json.NewEncoder(conn)
    decoder := json.NewDecoder(conn)
    
    encoder.Encode(request)
    
    var result mcp.CallToolResult
    decoder.Decode(&result)
    
    return &result, nil
}
```

### Putting it all together

```go
func main() {
    orchestrator := NewOrchestrator()
    
    // Load MCP server configurations
    configs := loadConfigs("mcp-servers.yaml")
    
    for _, cfg := range configs {
        // 1. Validate configuration
        if err := validateConfig(cfg); err != nil {
            log.Fatalf("Invalid config for %s: %v", cfg.Name, err)
        }
        
        // 2. Generate Nix config
        nixConfig, _ := generateVMConfig(MCPServerSpec{
            Hostname:   cfg.Name,
            Hypervisor: "firecracker",
            MemoryMB:   512,
            VCPUs:      2,
            VsockCID:   orchestrator.nextCID(),
            MCPCommand: cfg.Command,
        })
        
        // 3. Build VM image
        rootfsPath, _ := buildVM(context.Background(), nixConfig)
        
        // 4. Start VM
        vm := &MCPServerVM{
            socketPath: fmt.Sprintf("/tmp/fc-%s.sock", cfg.Name),
            vsockCID:   orchestrator.nextCID(),
        }
        vm.Create(context.Background(), rootfsPath)
        
        // 5. Register for routing
        orchestrator.RegisterServer(cfg.Name, vm)
        
        // 6. Inject secrets via vsock
        orchestrator.InjectSecrets(vm.vsockCID, cfg.Secrets)
    }
    
    // Start health checking
    go orchestrator.HealthCheckLoop()
    
    // Handle MCP client connections
    orchestrator.ServeMCP(":8080")
}
```

---

## Conclusion

This architecture provides **medium-security isolation** suitable for running MCP servers from partially-trusted sources. The key design decisions are:

- **Firecracker with Jailer** for minimal attack surface and fast boot times
- **Vsock-only communication** to eliminate network attack vectors between VMs
- **Read-only root filesystems** built deterministically with Nix
- **Runtime secret injection** to avoid persisting credentials
- **Go orchestrator** managing lifecycle, routing, and health

For higher security requirements, consider adding gVisor inside the VMs, implementing syscall filtering at the guest level, or using hardware security modules for secret storage. For lower security environments where boot time is critical, pre-built VM images with snapshot restore can achieve sub-50ms cold starts.

The combination of NixOS's declarative configuration and Firecracker's minimal hypervisor creates a reproducible, auditable system where security properties can be verified at build time rather than runtime.# Building secure MCP clients with microVM isolation

Running untrusted MCP servers safely requires **hardware-level isolation** that containers cannot provide. This primer presents a complete architecture combining Go orchestration, microvm.nix for declarative NixOS virtual machines, and the Firecracker VMM to create security boundaries between MCP servers and the host system. The approach draws from production patterns at AWS Lambda and Fly.io, adapted for the MCP ecosystem.

The key insight is that MCP servers, which can execute arbitrary tools and access external resources, represent a significant attack surface. By running each server in its own microVM with restricted networking and filesystem access, we achieve **defense-in-depth**: even if an MCP server is compromised, it cannot access other servers' data or the host system.

---

## Part 1: MCP client implementation fundamentals

### Protocol architecture

MCP uses **JSON-RPC 2.0** over two transport types: stdio for local servers (spawned as subprocesses) and Streamable HTTP for remote servers. The protocol follows a client-host-server model where the host application (like Claude Desktop) manages multiple client connections to different servers.

Every message must be UTF-8 encoded. Requests expect responses, while notifications are fire-and-forget:

```json
// Request
{"jsonrpc": "2.0", "id": 1, "method": "tools/call", 
 "params": {"name": "get_weather", "arguments": {"city": "London"}}}

// Response  
{"jsonrpc": "2.0", "id": 1, 
 "result": {"content": [{"type": "text", "text": "22°C"}]}}

// Notification (no id, no response expected)
{"jsonrpc": "2.0", "method": "notifications/progress", 
 "params": {"progress": 50, "total": 100}}
```

### The initialization handshake

Before any tool calls, clients must complete a three-step handshake that negotiates protocol version and capabilities:

```
Client                              Server
  │                                   │
  │──── initialize request ──────────▶│
  │                                   │
  │◀──── capabilities response ───────│
  │                                   │
  │──── initialized notification ────▶│
  │                                   │
  └───── operational phase ───────────┘
```

The initialize request declares the client's capabilities (sampling, roots), and the server responds with its capabilities (tools, resources, prompts, logging). Only after the client sends the `initialized` notification can regular operations begin.

### Building a Go MCP client

The official Go SDK provides a clean interface for client implementation:

```go
import (
    "context"
    "os/exec"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

func connectToServer(ctx context.Context, command string, args []string) (*mcp.Session, error) {
    client := mcp.NewClient(&mcp.Implementation{
        Name:    "my-orchestrator",
        Version: "1.0.0",
    }, nil)
    
    transport := &mcp.CommandTransport{
        Command: exec.Command(command, args...),
    }
    
    session, err := client.Connect(ctx, transport, nil)
    if err != nil {
        return nil, err
    }
    
    // Session is now initialized and ready for tool calls
    return session, nil
}

// Call a tool
func callTool(ctx context.Context, session *mcp.Session, name string, args map[string]any) error {
    result, err := session.CallTool(ctx, &mcp.CallToolParams{
        Name:      name,
        Arguments: args,
    })
    if err != nil {
        return err
    }
    // Process result.Content
    return nil
}
```

For stdio transport, the client spawns the server as a subprocess and communicates via newline-delimited JSON messages on stdin/stdout. The server's stderr can be used for logging without interfering with the protocol.

### Streamable HTTP transport

Remote servers use HTTP POST for requests, with the server responding either with `application/json` or `text/event-stream` (SSE) for streaming responses. Session management uses the `Mcp-Session-Id` header:

```go
// HTTP request headers
Accept: application/json, text/event-stream
MCP-Protocol-Version: 2025-06-18
Mcp-Session-Id: <session-id>  // After initialization
```

Security critical: Validate the `Origin` header to prevent DNS rebinding attacks, and bind to localhost only for local development servers.

---

## Part 2: MicroVM isolation with microvm.nix

### Why microVMs over containers

Containers share the host kernel, meaning a kernel vulnerability can lead to container escape. MicroVMs run their own kernel, reducing the attack surface to the hypervisor's device emulation code. Firecracker, written in **~50,000 lines of Rust** (versus QEMU's 1.4 million lines of C), provides a minimal attack surface while maintaining sub-200ms boot times.

microvm.nix provides a declarative, NixOS-native way to define and manage these VMs with reproducible builds.

### Hypervisor selection guide

| Hypervisor | Language | virtiofs | vsock | Security | Boot time | Best for |
|------------|----------|----------|-------|----------|-----------|----------|
| **Firecracker** | Rust | ❌ | ✅ | Excellent | ~125ms | Untrusted code |
| **cloud-hypervisor** | Rust | ✅ | ✅ | Very good | ~200ms | Needs file sharing |
| **QEMU** | C | ✅ | ✅ | Good | ~500ms | Development |

For MCP server isolation where servers need no shared filesystem, **Firecracker** is the clear choice. If servers need access to shared directories (like a codebase), **cloud-hypervisor** with virtiofs provides the best security/functionality tradeoff.

### Basic microvm.nix configuration

```nix
# flake.nix
{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    microvm.url = "github:astro/microvm.nix";
    microvm.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = { self, nixpkgs, microvm }: {
    nixosConfigurations.mcp-sandbox = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      modules = [
        microvm.nixosModules.microvm
        ./mcp-server-vm.nix
      ];
    };
  };
}
```

```nix
# mcp-server-vm.nix
{ pkgs, ... }: {
  networking.hostName = "mcp-sandbox";
  
  microvm = {
    hypervisor = "cloud-hypervisor";  # Or "firecracker"
    mem = 512;
    vcpu = 2;
    
    # Network: vsock only for controlled communication
    interfaces = [];  # No network interfaces
    vsock.cid = 100;  # Context ID for this VM
    
    # Filesystem: read-only root, persistent data volume
    storeDiskType = "squashfs";  # Compact, read-only
    storeOnDisk = true;          # Don't mount host /nix/store
    
    volumes = [{
      image = "data.img";
      mountPoint = "/data";
      size = 256;  # MiB
      autoCreate = true;
    }];
  };
  
  # Minimal services
  services.openssh.enable = false;
  
  # Install MCP server
  environment.systemPackages = with pkgs; [
    nodejs_20
    # Your MCP server package
  ];
  
  system.stateVersion = "24.05";
}
```

### Network isolation patterns

**Complete isolation with vsock** (recommended for untrusted servers):
```nix
microvm = {
  interfaces = [];     # No network
  vsock.cid = 100;     # vsock only
};
```

**TAP interface with host bridge** (for servers needing network access):
```nix
# In the VM configuration
microvm.interfaces = [{
  type = "tap";
  id = "vm-mcp-01";
  mac = "02:00:00:00:00:01";
}];

# On the host
systemd.network.netdevs."10-microvm".netdevConfig = {
  Kind = "bridge";
  Name = "microvm";
};

systemd.network.networks."11-microvm" = {
  matchConfig.Name = "vm-*";
  networkConfig.Bridge = "microvm";
};
```

### Filesystem security

The default read-only root filesystem provides strong integrity guarantees. For servers needing write access:

```nix
# Writable overlay for /nix/store (if server needs to install packages)
microvm = {
  writableStoreOverlay = "/nix/.rw-store";
  
  shares = [{
    proto = "virtiofs";
    tag = "rw-store";
    source = "/var/lib/microvms/my-server/rw-store";
    mountPoint = "/nix/.rw-store";
  }];
};
```

For cloud-hypervisor/QEMU, **virtiofs** provides faster shared directory access than 9p. Firecracker doesn't support filesystem shares—use volumes or include everything in the root image.

---

## Part 3: MCP server isolation patterns

### Container baseline (before migrating to VMs)

Many teams start with Docker isolation before graduating to VMs:

```yaml
# docker-compose.yml - hardened MCP server
services:
  mcp-server:
    image: my-mcp-server:latest
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    read_only: true
    tmpfs:
      - /tmp
    networks:
      - mcp-internal
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
          pids: 100

networks:
  mcp-internal:
    internal: true  # No external access
```

This provides namespace isolation but shares the host kernel. For stronger isolation, use **gVisor** (`--runtime runsc`) or migrate to VMs.

### Credential handling architecture

Never store credentials in plaintext configuration files. Use a layered approach:

**Level 1 - Environment injection** (development only):
```json
{
  "mcpServers": {
    "github": {
      "command": "node",
      "args": ["server.js"],
      "env": {"GITHUB_TOKEN": "${GITHUB_TOKEN}"}
    }
  }
}
```

**Level 2 - Secrets manager integration** (recommended):
```bash
# Using 1Password CLI
op run --env-file=mcp.env -- node mcp-server.js

# mcp.env references vault paths
GITHUB_TOKEN=op://vault/github/token
```

**Level 3 - Runtime injection via vsock** (for VM isolation):
```go
// Orchestrator retrieves secret and sends via vsock
secret := vault.GetSecret("mcp-server-1/github-token")
conn, _ := vsock.Dial(vmCID, secretPort, nil)
conn.Write([]byte(secret))
```

The VM's init process reads from vsock and sets environment variables without persisting to disk.

### Network security for MCP servers

Implement egress filtering to prevent data exfiltration:

```nix
# In VM configuration - firewall rules
networking.firewall = {
  enable = true;
  allowedTCPPorts = [];  # No inbound
  
  # Egress filtering via nftables
  extraCommands = ''
    # Allow only specific API endpoints
    iptables -A OUTPUT -d api.github.com -p tcp --dport 443 -j ACCEPT
    iptables -A OUTPUT -d api.openai.com -p tcp --dport 443 -j ACCEPT
    iptables -A OUTPUT -j DROP
  '';
};
```

For complete isolation, run with no network interfaces and proxy all requests through the orchestrator via vsock.

---

## Part 4: Go binary orchestration

### Firecracker Go SDK fundamentals

The firecracker-go-sdk provides the cleanest interface for VM management:

```go
import (
    firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
    models "github.com/firecracker-microvm/firecracker-go-sdk/client/models"
)

type MCPServerVM struct {
    machine    *firecracker.Machine
    socketPath string
    vsockCID   uint32
}

func (m *MCPServerVM) Create(ctx context.Context, rootfsPath string) error {
    cfg := firecracker.Config{
        SocketPath:      m.socketPath,
        KernelImagePath: "/var/lib/firecracker/vmlinux",
        KernelArgs:      "console=ttyS0 reboot=k panic=1 init=/init",
        Drives: []models.Drive{{
            DriveID:      firecracker.String("rootfs"),
            PathOnHost:   firecracker.String(rootfsPath),
            IsRootDevice: firecracker.Bool(true),
            IsReadOnly:   firecracker.Bool(true),
        }},
        MachineCfg: models.MachineConfiguration{
            VcpuCount:  firecracker.Int64(2),
            MemSizeMib: firecracker.Int64(512),
        },
        VsockDevices: []firecracker.VsockDevice{{
            Path: m.socketPath + ".vsock",
            CID:  m.vsockCID,
        }},
        // Jailer for additional isolation
        JailerCfg: &firecracker.JailerConfig{
            UID:           firecracker.Int(65534),  // nobody
            GID:           firecracker.Int(65534),
            ChrootBaseDir: "/srv/jailer",
            Daemonize:     false,
        },
    }
    
    cmd := firecracker.VMCommandBuilder{}.
        WithSocketPath(m.socketPath).
        WithBin("/usr/bin/firecracker").
        Build(ctx)
    
    machine, err := firecracker.NewMachine(ctx, cfg, firecracker.WithProcessRunner(cmd))
    if err != nil {
        return err
    }
    
    m.machine = machine
    return machine.Start(ctx)
}
```

### Vsock communication patterns

Vsock enables direct host-guest communication without network configuration:

```go
import "github.com/mdlayher/vsock"

// Host side: listen for connections from VMs
func (o *Orchestrator) listenVsock(port uint32) error {
    listener, err := vsock.Listen(port, nil)
    if err != nil {
        return err
    }
    
    for {
        conn, err := listener.Accept()
        if err != nil {
            continue
        }
        
        // Get VM's CID to identify which server connected
        addr := conn.RemoteAddr().(*vsock.Addr)
        vmCID := addr.ContextID
        
        go o.handleMCPConnection(conn, vmCID)
    }
}

// Guest side: connect to host
func connectToHost(port uint32) (net.Conn, error) {
    return vsock.Dial(vsock.Host, port, nil)  // CID 2 = host
}
```

For MCP protocol forwarding, run a proxy on the host that accepts vsock connections and translates to/from the client:

```go
func (o *Orchestrator) handleMCPConnection(vsockConn net.Conn, vmCID uint32) {
    // Find the MCP session for this VM
    session := o.sessions[vmCID]
    
    // Bidirectional copy - MCP messages flow through vsock
    go io.Copy(session.stdin, vsockConn)   // Client -> VM
    io.Copy(vsockConn, session.stdout)     // VM -> Client
}
```

### Lifecycle management

```go
type VMManager struct {
    vms       map[uint32]*MCPServerVM
    mu        sync.RWMutex
    healthTick *time.Ticker
}

func (m *VMManager) HealthCheck(ctx context.Context) {
    for range m.healthTick.C {
        m.mu.RLock()
        for cid, vm := range m.vms {
            go func(cid uint32, vm *MCPServerVM) {
                ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
                defer cancel()
                
                conn, err := vsock.DialContext(ctx, cid, healthPort, nil)
                if err != nil {
                    log.Printf("VM %d unhealthy: %v", cid, err)
                    m.RestartVM(cid)
                    return
                }
                conn.Close()
            }(cid, vm)
        }
        m.mu.RUnlock()
    }
}

func (m *VMManager) GracefulShutdown(ctx context.Context, cid uint32, timeout time.Duration) error {
    vm := m.vms[cid]
    
    // Send shutdown signal via vsock
    conn, _ := vsock.Dial(cid, shutdownPort, nil)
    conn.Write([]byte("shutdown"))
    conn.Close()
    
    // Wait with timeout
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()
    
    done := make(chan struct{})
    go func() {
        vm.machine.Wait(ctx)
        close(done)
    }()
    
    select {
    case <-done:
        return nil
    case <-ctx.Done():
        return vm.machine.StopVMM()  // Force kill
    }
}
```

### Dynamic Nix configuration generation

For programmatic VM creation, generate Nix configurations from Go:

```go
const vmTemplate = `
{ pkgs, ... }: {
  networking.hostName = "{{.Hostname}}";
  microvm = {
    hypervisor = "{{.Hypervisor}}";
    mem = {{.MemoryMB}};
    vcpu = {{.VCPUs}};
    vsock.cid = {{.VsockCID}};
    interfaces = [];
  };
  
  environment.systemPackages = [
    {{range .Packages}}pkgs.{{.}}
    {{end}}
  ];
  
  systemd.services.mcp-server = {
    wantedBy = ["multi-user.target"];
    serviceConfig = {
      ExecStart = "{{.MCPCommand}}";
      Environment = [
        {{range $k, $v := .Environment}}"{{$k}}={{$v}}"
        {{end}}
      ];
    };
  };
  
  system.stateVersion = "24.05";
}
`

func generateVMConfig(spec MCPServerSpec) (string, error) {
    tmpl := template.Must(template.New("vm").Parse(vmTemplate))
    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, spec); err != nil {
        return "", err
    }
    return buf.String(), nil
}

func buildVM(ctx context.Context, nixConfig string) (string, error) {
    // Write config to temp file
    configPath := filepath.Join(os.TempDir(), "vm-config.nix")
    os.WriteFile(configPath, []byte(nixConfig), 0644)
    
    // Build with nix
    cmd := exec.CommandContext(ctx, "nix", "build", 
        "--out-link", "/tmp/result",
        "-f", configPath)
    return "/tmp/result", cmd.Run()
}
```

---

## Part 5: Complete architecture

### System overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     Go Orchestrator (Host)                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ MCP Client   │  │ VM Manager   │  │ Config Gen   │          │
│  │ (Sessions)   │  │ (Lifecycle)  │  │ (Nix Build)  │          │
│  └──────┬───────┘  └──────┬───────┘  └──────────────┘          │
│         │                 │                                      │
│         │     vsock       │    Firecracker                      │
│         │   connections   │    processes                         │
├─────────┴─────────────────┴──────────────────────────────────────┤
│                     KVM / Host Kernel                            │
├──────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │  MicroVM 1  │  │  MicroVM 2  │  │  MicroVM 3  │              │
│  │ ┌─────────┐ │  │ ┌─────────┐ │  │ ┌─────────┐ │              │
│  │ │MCP Srv A│ │  │ │MCP Srv B│ │  │ │MCP Srv C│ │              │
│  │ └─────────┘ │  │ └─────────┘ │  │ └─────────┘ │              │
│  │  CID: 100   │  │  CID: 101   │  │  CID: 102   │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
└──────────────────────────────────────────────────────────────────┘
```

### Security boundaries and trust model

The architecture provides **three isolation layers**:

1. **KVM hardware virtualization**: Primary boundary. Guest cannot access host memory or other guests.
2. **Firecracker's minimal VMM**: Only 5 emulated devices, ~40 allowed syscalls via seccomp.
3. **Jailer sandbox**: Firecracker process runs in chroot with dropped privileges.

Trust levels:
- **Trusted**: Host kernel, Go orchestrator, Nix build system
- **Semi-trusted**: Firecracker VMM process (in jail)
- **Untrusted**: MCP server code inside VM

### Configuration flow

```
User Config (YAML)          Security Validation          Nix Generation
     │                            │                           │
     ▼                            ▼                           ▼
┌──────────┐              ┌──────────────┐            ┌──────────────┐
│ command  │──validate───▶│ Allowlisted  │──generate─▶│ microvm.nix  │
│ env vars │              │ commands only│            │ expression   │
│ resources│              │ No host paths│            │              │
└─────────┘              └──────────────┘            └──────────────┘
                                                            │
                                                     nix build
                                                            │
                                                            ▼
                               ┌──────────────────────────────────┐
                               │ VM rootfs (squashfs, read-only)  │
                               │ + runtime secrets via vsock      │
                               └──────────────────────────────────┘
```

### MCP request routing

```go
type Orchestrator struct {
    sessions   map[string]*mcp.Session    // External client sessions
    vms        map[uint32]*MCPServerVM    // VM by CID
    serverToVM map[string]uint32          // MCP server name -> VM CID
}

func (o *Orchestrator) RouteToolCall(ctx context.Context, serverName string, 
    tool string, args map[string]any) (*mcp.CallToolResult, error) {
    
    cid, ok := o.serverToVM[serverName]
    if !ok {
        return nil, fmt.Errorf("unknown server: %s", serverName)
    }
    
    // Connect to MCP server in VM via vsock
    conn, err := vsock.DialContext(ctx, cid, mcpPort, nil)
    if err != nil {
        return nil, fmt.Errorf("VM unreachable: %w", err)
    }
    defer conn.Close()
    
    // Forward the tool call
    request := mcp.CallToolRequest{
        Method: "tools/call",
        Params: mcp.CallToolParams{Name: tool, Arguments: args},
    }
    
    // JSON-RPC over vsock
    encoder := json.NewEncoder(conn)
    decoder := json.NewDecoder(conn)
    
    encoder.Encode(request)
    
    var result mcp.CallToolResult
    decoder.Decode(&result)
    
    return &result, nil
}
```

### Putting it all together

```go
func main() {
    orchestrator := NewOrchestrator()
    
    // Load MCP server configurations
    configs := loadConfigs("mcp-servers.yaml")
    
    for _, cfg := range configs {
        // 1. Validate configuration
        if err := validateConfig(cfg); err != nil {
            log.Fatalf("Invalid config for %s: %v", cfg.Name, err)
        }
        
        // 2. Generate Nix config
        nixConfig, _ := generateVMConfig(MCPServerSpec{
            Hostname:   cfg.Name,
            Hypervisor: "firecracker",
            MemoryMB:   512,
            VCPUs:      2,
            VsockCID:   orchestrator.nextCID(),
            MCPCommand: cfg.Command,
        })
        
        // 3. Build VM image
        rootfsPath, _ := buildVM(context.Background(), nixConfig)
        
        // 4. Start VM
        vm := &MCPServerVM{
            socketPath: fmt.Sprintf("/tmp/fc-%s.sock", cfg.Name),
            vsockCID:   orchestrator.nextCID(),
        }
        vm.Create(context.Background(), rootfsPath)
        
        // 5. Register for routing
        orchestrator.RegisterServer(cfg.Name, vm)
        
        // 6. Inject secrets via vsock
        orchestrator.InjectSecrets(vm.vsockCID, cfg.Secrets)
    }
    
    // Start health checking
    go orchestrator.HealthCheckLoop()
    
    // Handle MCP client connections
    orchestrator.ServeMCP(":8080")
}
```

---

## Conclusion

This architecture provides **medium-security isolation** suitable for running MCP servers from partially-trusted sources. The key design decisions are:

- **Firecracker with Jailer** for minimal attack surface and fast boot times
- **Vsock-only communication** to eliminate network attack vectors between VMs
- **Read-only root filesystems** built deterministically with Nix
- **Runtime secret injection** to avoid persisting credentials
- **Go orchestrator** managing lifecycle, routing, and health

For higher security requirements, consider adding gVisor inside the VMs, implementing syscall filtering at the guest level, or using hardware security modules for secret storage. For lower security environments where boot time is critical, pre-built VM images with snapshot restore can achieve sub-50ms cold starts.

The combination of NixOS's declarative configuration and Firecracker's minimal hypervisor creates a reproducible, auditable system where security properties can be verified at build time rather than runtime.
