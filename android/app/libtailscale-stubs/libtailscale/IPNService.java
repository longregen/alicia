package libtailscale;

public interface IPNService {
    String id();
    boolean protect(int fd);
    VPNServiceBuilder newBuilder();
    void close();
    void disconnectVPN();
    void updateVpnStatus(boolean status);
}
