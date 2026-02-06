package libtailscale;

public interface LocalAPIResponse {
    long statusCode();
    byte[] bodyBytes() throws Exception;
    InputStream bodyInputStream();
}
