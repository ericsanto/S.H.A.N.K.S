package enum

type TableFormat string

const (
	Iceberg TableFormat = "iceberg"
)

func (tf TableFormat) String() string {
	return string(tf)
}
