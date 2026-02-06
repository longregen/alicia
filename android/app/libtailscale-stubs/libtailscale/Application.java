package libtailscale;

public interface Application {
    LocalAPIResponse callLocalAPI(long timeoutMillis, String method, String endpoint, InputStream body) throws Exception;
    void notifyPolicyChanged();
    NotificationManager watchNotifications(long mask, NotificationCallback cb);
}
