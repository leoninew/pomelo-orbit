ALTER TABLE gateway_config
    DROP CONSTRAINT IF EXISTS gateway_config_application_id_fkey,
    ADD CONSTRAINT gateway_config_application_id_fkey
        FOREIGN KEY (application_id) REFERENCES application(id);

ALTER TABLE gateway_acme_profile_version
    DROP CONSTRAINT IF EXISTS gateway_acme_profile_version_version_id_fkey,
    ADD CONSTRAINT gateway_acme_profile_version_version_id_fkey
        FOREIGN KEY (version_id) REFERENCES version(id);

ALTER TABLE service
    DROP CONSTRAINT IF EXISTS service_application_id_fkey,
    ADD CONSTRAINT service_application_id_fkey
        FOREIGN KEY (application_id) REFERENCES application(id);

ALTER TABLE service_component
    DROP CONSTRAINT IF EXISTS service_component_source_version_component_id_fkey,
    ADD CONSTRAINT service_component_source_version_component_id_fkey
        FOREIGN KEY (source_version_component_id) REFERENCES version_component(id);
