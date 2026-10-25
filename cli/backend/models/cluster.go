package models

type Cluster struct {
	Namenode  Namenode    `yaml:"namenode"`
	Datanodes []*Datanode `yaml:"datanodes"`
	Lakehouse Lakehouse   `yaml:"lakehouse"`
	Catalog   Catalog     `yaml:"catalog"`
	Storage   Storage     `yaml:"storage"`
	Bi        Bi          `yaml:"bi"`
}

type Lakehouse struct {
	Enabled     bool   `yaml:"enabled"`
	TableFormat string `yaml:"table_format" default:"iceberg"`
}

type Catalog struct {
	Type string `yaml:"type" default:"hadoop"`
}

type Storage struct {
	Type      string `yaml:"type" default:"hdfs"`
	Warehouse string `yaml:"warehouse" default:"hdfs://master:9000/lakehouse"`
}

type Bi struct {
	Enabled bool   `yaml:"enabled" default:"false"`
	Tool    string `yaml:"tool" default:"superset"`
	Node    string `yaml:"node" default:"master"`
}
