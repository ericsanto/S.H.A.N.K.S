package models

type Cluster struct {
	Namenode  Namenode    `yaml:"namenode"`
	Datanodes []*Datanode `yaml:"datanodes"`
}
