package dev.ggid.sdk;

/**
 * Exception thrown when a GGID API call fails.
 */
public class GGIDException extends Exception {

    private final int statusCode;
    private final String code;

    public GGIDException(int statusCode, String message, String code) {
        super(message);
        this.statusCode = statusCode;
        this.code = code;
    }

    public int getStatusCode() {
        return statusCode;
    }

    public String getCode() {
        return code;
    }

    public boolean isNotFound() {
        return statusCode == 404;
    }

    public boolean isUnauthorized() {
        return statusCode == 401;
    }

    public boolean isForbidden() {
        return statusCode == 403;
    }

    public boolean isConflict() {
        return statusCode == 409;
    }

    public boolean isRateLimited() {
        return statusCode == 429;
    }

    @Override
    public String toString() {
        return "GGIDException{" +
                "statusCode=" + statusCode +
                ", code='" + code + '\'' +
                ", message='" + getMessage() + '\'' +
                '}';
    }
}
