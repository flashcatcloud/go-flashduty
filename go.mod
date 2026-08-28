module github.com/flashcatcloud/go-flashduty

go 1.24

// v0.14.3 dropped the binary-download (CSV export) methods and emitted a
// no-arg multipart upload method; both are fixed in v0.14.4.
retract v0.14.3
