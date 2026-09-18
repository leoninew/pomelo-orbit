-- Historical incomplete Gateway seed cleanup. Deployment resources are created by Project initialization.
DELETE FROM `route` WHERE `id` = '01M01ZNW6CPQCB7P5PN669HWJ9';
DELETE FROM `service_component` WHERE `service_id` = '01M01RHDXW3ZXC7YKNT54RWM1M';
DELETE FROM `service` WHERE `application_id` = '01M01MP0950ECGK2DS1FWYNC0B';
DELETE FROM `application` WHERE `id` = '01M01MP0950ECGK2DS1FWYNC0B';
