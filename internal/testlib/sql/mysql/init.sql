CREATE TABLE IF NOT EXISTS Table1 (
    Id varchar(26) NOT NULL,
    CreateAt bigint(20) DEFAULT NULL,
    Name varchar(64) DEFAULT NULL,
    Description text,
    PRIMARY KEY (Id),
    UNIQUE KEY Name (Name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS Table2 (
    Id varchar(26) NOT NULL,
    AnotherId varchar(26) NOT NULL,
    IsActive tinyint(4),
    Props JSON,
    PRIMARY KEY (Id, AnotherId)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
