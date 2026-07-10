package dev.ggid.sdk;

/** Thrown when GGID API calls fail. */
public class GGIDException extends Exception {
    public GGIDException(String message) {
        super(message);
    }

    public GGIDException(String message, Throwable cause) {
        super(message, cause);
    }
}
