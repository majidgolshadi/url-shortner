DROP TABLE IF EXISTS `node_range_journal`;

CREATE TABLE `node_range_journal` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `node_id` VARCHAR(32) NOT NULL,
    `start` INT UNSIGNED,
    `end` INT UNSIGNED,
    PRIMARY KEY (`id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

# ------------------------------------------------------------

DROP TABLE IF EXISTS `nodes_coodination_keys`;

CREATE TABLE `nodes_coordination_keys` (
    `key_id` VARCHAR(32) NOT NULL,
    `value` VARCHAR(64) NOT NULL,
    `version` INT UNSIGNED,
    PRIMARY KEY (`key_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

# ------------------------------------------------------------

DROP TABLE IF EXISTS `customer`;

CREATE TABLE `customer` (
    `id`         VARCHAR(36)  NOT NULL,
    `auth_token` VARCHAR(64)  NOT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_customer_auth_token` (`auth_token`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

# ------------------------------------------------------------

DROP TABLE IF EXISTS `url_token`;

CREATE TABLE `url_token` (
   `token`       VARCHAR(100) NOT NULL,
   `url`         VARCHAR(100) NOT NULL,
   `headers`     JSON         DEFAULT NULL,
   `og_html`     TEXT         DEFAULT NULL,
   `customer_id` VARCHAR(36)  NOT NULL DEFAULT '',
   PRIMARY KEY (`token`),
   KEY `idx_url_token_customer_id` (`customer_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

# ------------------------------------------------------------

DELIMITER //
CREATE TRIGGER check_version_before_insert
    BEFORE INSERT ON nodes_coordination_keys
    FOR EACH ROW
BEGIN
    DECLARE existing_version INT;

    SELECT version INTO existing_version
    FROM nodes_coordination_keys
    WHERE key_id = NEW.key_id;

    IF existing_version IS NOT NULL AND NEW.version <= existing_version THEN
        SIGNAL SQLSTATE '45000'
        SET MESSAGE_TEXT = 'Error: The requested version value must be more than the existing version value.';
END IF;
END;//
DELIMITER ;