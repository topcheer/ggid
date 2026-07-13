use thiserror::Error;

#[derive(Error, Debug)]
pub enum GGIDError {
    #[error("invalid token: {0}")]
    InvalidToken(String),

    #[error("token expired")]
    TokenExpired,

    #[error("permission denied: {0}:{1}")]
    PermissionDenied(String, String),

    #[error("HTTP error: {0}")]
    Http(#[from] reqwest::Error),

    #[error("JSON error: {0}")]
    Json(#[from] serde_json::Error),

    #[error("JWT error: {0}")]
    Jwt(#[from] jsonwebtoken::errors::Error),

    #[error("API error: status={status}, body={body}")]
    Api { status: u16, body: String },

    #[error("{0}")]
    Other(String),
}

impl From<GGIDError> for std::io::Error {
    fn from(e: GGIDError) -> Self {
        std::io::Error::new(std::io::ErrorKind::Other, e.to_string())
    }
}
