package libtailscale;

public interface VPNServiceBuilder {
    void setMTU(int mtu) throws Exception;
    void addDNSServer(String server) throws Exception;
    void addSearchDomain(String domain) throws Exception;
    void addRoute(String addr, int prefix) throws Exception;
    void excludeRoute(String addr, int prefix) throws Exception;
    void addAddress(String addr, int prefix) throws Exception;
    ParcelFileDescriptor establish() throws Exception;
}
