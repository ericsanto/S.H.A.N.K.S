package models

import (
	"fmt"
	"strconv"
)

type Datanode struct {
	User               string `yaml:"user"`
	Name               string `yaml:"name"`
	IP                 string `yaml:"ip"`
	YarnLimit          string `yaml:"yarn_limit"`
	SparkLimit         string `yaml:"spark_limit"`
	YarnLimitCPUVcores string `yaml:"cpu"`
	GPU                bool   `yaml:"gpu;default:false"`
}

func (d *Datanode) ConvertToYarnLimit() (float64, error) {

	intYarnLimit, err := strconv.ParseFloat(d.YarnLimit, 64)
	if err != nil {
		return 0, err
	}

	byteYarnLimit := intYarnLimit * 1000

	return byteYarnLimit, nil

}

func (d *Datanode) VerifiyValueMinYarnRequirements() error {

	yarnLimit, err := d.ConvertToYarnLimit()
	if err != nil {
		return err
	}

	if yarnLimit < 1024 {
		return fmt.Errorf("o limite de memória do Yarn para o Datanode %s é menor que o mínimo requerido (1024 MB)", d.Name)
	}

	return nil
}

func (d *Datanode) ConvertCPULimit() (*int, error) {

	cpuLimit, err := strconv.Atoi(d.YarnLimitCPUVcores)

	if err != nil {
		return nil, err
	}

	return &cpuLimit, nil
}
