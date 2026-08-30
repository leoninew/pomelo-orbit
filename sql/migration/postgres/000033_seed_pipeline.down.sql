-- Remove rows owned by the pipeline seed.

DELETE FROM "pipeline_stage_reference" WHERE ("id" = '01KZGBG9NT6NCK8AT6H6ENV874') OR ("id" = '01KZGBGDK1G249681EVBDA9035') OR ("id" = '4smvyi2oq4n2kzrio4zmcqykge') OR ("id" = 'cirr32qhvah2rbv2obb76f7fte') OR ("id" = 'vb37nbzugq6pljhkbxm3hiij24') OR ("id" = 'yrkdm4fc4wlupvd3ne6ea2ywpe');
DELETE FROM "pipeline_stage" WHERE ("id" = '01KNRANZDR4PASATAXKTBBTRX9') OR ("id" = '01KNRDSSJ7RNND7110175N4NR2') OR ("id" = '01KNRKNAHG3EBS07VBK2YY5ZQN') OR ("id" = '01KRCWNJVA1DM02TJXZ4STJD01') OR ("id" = '01KRCWNJVA1DM02TJXZ4STJD06');
DELETE FROM "pipeline" WHERE ("id" = '01KNVEJPWVK757139NMNNNCEFE') OR ("id" = '01KZG83K2MXG08EJ6G48SG38B3');
