ALTER TABLE totp_configurations RENAME _bkp_DOWN_V0002_totp_configurations;
ALTER TABLE webauthn_devices RENAME _bkp_DOWN_V0002_webauthn_devices;

CREATE TABLE totp_configurations (
    id INTEGER AUTO_INCREMENT,
    username VARCHAR(100) NOT NULL,
    issuer VARCHAR(100),
    algorithm VARCHAR(6) NOT NULL DEFAULT 'SHA1',
    digits INTEGER NOT NULL DEFAULT 6,
    period INTEGER NOT NULL DEFAULT 30,
    secret BLOB NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY (username)
) ENGINE InnoDB DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_520_ci;

INSERT INTO totp_configurations (id, username, issuer, algorithm, digits, period, secret)
SELECT id, username, issuer, algorithm, digits, period, secret
FROM _bkp_DOWN_V0002_totp_configurations;

CREATE TABLE u2f_devices (
    id INTEGER AUTO_INCREMENT,
    username VARCHAR(100) NOT NULL,
    description VARCHAR(30) NOT NULL DEFAULT 'Primary',
    key_handle BLOB NOT NULL,
    public_key BLOB NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY (username, description)
) ENGINE InnoDB DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_520_ci;

INSERT INTO u2f_devices (id, username, description, key_handle, public_key)
SELECT id, username, description, FROM_BASE64(kid), public_key
FROM _bkp_DOWN_V0002_webauthn_devices
WHERE attestation_type = 'fido-u2f';

UPDATE user_preferences
SET second_factor_method = 'u2f'
WHERE second_factor_method = 'webauthn';

DROP TABLE _bkp_DOWN_V0002_totp_configurations;
DROP TABLE _bkp_DOWN_V0002_webauthn_devices;
