package models

//go:generate xsdgen -o usd.go -pkg models USD-schema-main.xsd USD-Rel-12-schema-snippet.xsd   USD-Rel-15-schema-snippet.xsd   USD-Rel-7-schema-snippet.xsd    USD-Rel-9-schema-snippet.xsd    schemaversion.xsd USD-Rel-14-schema-snippet.xsd   USD-Rel-16-schema-snippet.xsd   USD-Rel-8-schema-snippet.xsd
//go:generate xsdgen -o schedule.go -pkg models Schedule-Description-Main.xsd           Schedule-Rel-11-schema-snippet.xsd      Schedule-Rel-12-schema-snippet.xsd      schema-version.xsd
//go:generate xsdgen -o filter.go -pkg models FilterDescription.xsd Filter-Rel-12-schema-snippet.xsd schema-version.xsd
//go:generate xsdgen -o mlp.go -pkg models -ns MLP_SVC_RESULT_310.dtd mlp_svc_result_310.xsd mlp_svc_init_310.xsd
//go:generate xsdgen -o security.go -pkg models security.xsd
//go:generate xsdgen -o apd.go -pkg models adpd-rel-12-extension.xsd       adpd-rel-13-extension.xsd       adpd-rel14-extension.xsd        associatedprocedure.xsd schema-version.xsd
