CREATE TABLE IF NOT EXISTS metrics
(
    id      SERIAL PRIMARY KEY,
    m_name  VARCHAR(50) UNIQUE,
    m_value VARCHAR(20),
    m_type  VARCHAR(20)
);