package libtailscale;

public interface InputStream {
    byte[] read() throws Exception;
    void close() throws Exception;
}
