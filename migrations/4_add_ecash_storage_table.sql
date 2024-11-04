-- +goose Up
CREATE TABLE IF NOT EXISTS cashu_pubkey(
    version INTEGER PRIMARY KEY, 
    active BOOL NOT NULL, 
    created_at INTEGER not NULL
);

CREATE TABLE IF NOT EXISTS stored_proofs(
	amount INTEGER NOT NULL,
	id text NOT NULL,
	secret text NOT NULL UNIQUE,
	C text NOT NULL UNIQUE,
	witness text,
	redeemed bool NOT NULL,
	spent bool NOT NULL DEFAULT FALSE,
    created_at INTEGER not NULL,
    pubkey_version INTEGER not NULL,
    FOREIGN KEY (pubkey_version) REFERENCES cashu_pubkey(version)
);

CREATE TABLE IF NOT EXISTS trusted_mints(
    url TEXT NOT NULL,
    created_at INTEGER not NULL
);



-- +goose Down
DROP TABLE IF EXISTS stored_proofs;
DROP TABLE IF EXISTS trusted_mints;
DROP TABLE IF EXISTS cashu_pubkey;

