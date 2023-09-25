BEGIN;
ALTER TABLE accounts ADD COLUMN channel_id integer;
ALTER TABLE accounts ADD CONSTRAINT accounts_channel_id_fkey FOREIGN KEY (channel_id) REFERENCES channels (id) ON UPDATE RESTRICT ON DELETE RESTRICT;
UPDATE accounts a SET channel_id=t.channel_id FROM accounts a1 LEFT JOIN transactions t ON t.account_id = a1.id WHERE a.id=a1.id;
ALTER TABLE accounts ALTER COLUMN channel_id SET not null;
ALTER TABLE routes ALTER COLUMN account_id DROP not null;

ALTER TABLE transactions DROP COLUMN channel_id;
ALTER TABLE routes DROP COLUMN channel_id;
COMMIT;
