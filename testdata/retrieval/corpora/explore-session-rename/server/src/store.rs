use time::OffsetDateTime;
use uuid::Uuid;

pub struct Session {
    pub id: Uuid,
    pub user_id: Uuid,
    pub expires_at: OffsetDateTime,
}

pub fn new_session(user_id: Uuid, ttl: time::Duration) -> Session {
    Session {
        id: Uuid::new_v4(),
        user_id,
        expires_at: OffsetDateTime::now_utc() + ttl,
    }
}
