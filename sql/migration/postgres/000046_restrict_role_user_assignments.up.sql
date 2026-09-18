ALTER TABLE user_role
    DROP CONSTRAINT IF EXISTS user_role_role_id_fkey,
    ADD CONSTRAINT user_role_role_id_fkey
        FOREIGN KEY (role_id) REFERENCES role(id);
