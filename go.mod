module github.com/lyhna-ai/lyhna-go

go 1.21

retract v0.1.0 // contract drift: exposed caller-side authority_tier, missing intent_version, used non-canonical payload wire key
