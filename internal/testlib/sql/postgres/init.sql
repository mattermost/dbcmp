CREATE TABLE IF NOT EXISTS Table1 (
    Id VARCHAR(26) PRIMARY KEY,
    CreateAt bigint,
    Name VARCHAR(64),
    Description VARCHAR(1000),
    UNIQUE(name)
);

CREATE TABLE IF NOT EXISTS Table2 (
    Id varchar(26) PRIMARY KEY,
    IsActive boolean,
    Props JSONB
);
