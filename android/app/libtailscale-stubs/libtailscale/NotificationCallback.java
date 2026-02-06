package libtailscale;

public interface NotificationCallback {
    void onNotify(byte[] notification) throws Exception;
}
