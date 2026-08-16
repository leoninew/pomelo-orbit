-- Remove rows owned by the gateway seed.

DELETE FROM `route` WHERE (`id` = '01M01ZNW6CPQCB7P5PN669HWJ9');
DELETE FROM `service_component` WHERE (`id` = '01M01RHDXW3ZXC7YKNT8GDNQNB');
DELETE FROM `service` WHERE (`id` = '01M01RHDXW3ZXC7YKNT54RWM1M');
DELETE FROM `version_component_mount` WHERE (`component_id` = '01M01MP096R73Z3MG28P91CSPN' AND `position` = 0) OR (`component_id` = '01M01MP096R73Z3MG28P91CSPN' AND `position` = 1) OR (`component_id` = '01M01MP096R73Z3MG28P91CSPN' AND `position` = 2);
DELETE FROM `version_component_endpoint` WHERE (`component_id` = '01M01MP096R73Z3MG28P91CSPN' AND `protocol` = 'http' AND `container_port` = 8080) OR (`component_id` = '01M01MP096R73Z3MG28P91CSPN' AND `protocol` = 'tcp' AND `container_port` = 80) OR (`component_id` = '01M01MP096R73Z3MG28P91CSPN' AND `protocol` = 'tcp' AND `container_port` = 443);
DELETE FROM `version_component` WHERE (`id` = '01M01MP096R73Z3MG28P91CSPN');
DELETE FROM `gateway_config` WHERE (`application_id` = '01M01MP0950ECGK2DS1FWYNC0B');
DELETE FROM `version` WHERE (`id` = '01M01MP0950ECGK2DS1J4N4P50');
DELETE FROM `application` WHERE (`id` = '01M01MP0950ECGK2DS1FWYNC0B');
