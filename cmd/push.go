// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/shirou/gopsutil/process"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

var DryRun bool

var command = func(cmd *cobra.Command, args []string) (err error) {
	if len(args) < 1 {
		return errors.New("push requires `processname`")
	}

	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)
	log.Info("dryrun=", DryRun)
	pids, err := process.Pids()

	if err != nil {
		log.Error("cannot read pids", err.Error())
	}

	procNames := []string{}

	for _, pid := range pids {
		proc := process.Process{
			Pid: pid,
		}

		procName, err := proc.Name()
		if err != nil {
			log.Error("error getting name", err.Error())
		}
		procNames = append(procNames, procName)
	}

	log.Info("there are ", len(procNames), " processes running")
	processToMonitor := args[0]
	if len(intersect(procNames, []string{processToMonitor})) > 0 {
		log.Info(processToMonitor, " is running")
		pushMetric(processToMonitor, 1)
	} else {
		log.Error(processToMonitor, " is not running")
	}
	return
}

func pushMetric(procName string, count float64) {
	sess := session.Must(session.NewSession())
	awsRegion := "us-east-1"
	svc := cloudwatch.New(sess, &aws.Config{Region: aws.String(awsRegion)})
	value := count
	unit := cloudwatch.StandardUnitCount
	params := cloudwatch.PutMetricDataInput{
		MetricData: []*cloudwatch.MetricDatum{
			{
				MetricName: aws.String(procName),
				Dimensions: []*cloudwatch.Dimension{
					{
						Name:  aws.String("count"),
						Value: aws.String("count"),
					},
				},
				Unit:  &unit,
				Value: &value,
			},
		},
		Namespace: aws.String("process"),
	}

	if !DryRun {
		log.Info("putting metrics...")
		_, err := svc.PutMetricData(&params)
		if err != nil {
			log.Error("error pushing metrics ", err.Error())
		}
		log.Info("done")
	} else {
		log.Info("will put count=", count, " processname=", procName)
	}
}

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "push metrics to cloudwatch",
	Long:  "",
	RunE: command,
}

func init() {
	RootCmd.AddCommand(pushCmd)
	pushCmd.Flags().BoolVarP(&DryRun, "dryrun", "d", false, "dry run")
}

func intersect(arrs ...[]string) []string {
	tempMaps := make([]map[string]int, len(arrs))
	intersectionMap := make(map[string]int)

	for i, arr := range arrs {
		tempMaps[i] = make(map[string]int)
		for _, val := range arr {
			if _, ok := tempMaps[i][val]; !ok {
				tempMaps[i][val] = 1
			}
		}
	}

	for _, tempMap := range tempMaps {
		for k := range tempMap {
			if _, ok := intersectionMap[k]; ok {
				intersectionMap[k]++
			} else {
				intersectionMap[k] = 1
			}
		}
	}

	intersection := []string{}
	for k, v := range intersectionMap {
		if v > 1 {
			intersection = append(intersection, k)
		}
	}
	return intersection
}

func contains(values []string, value string) bool {
	for _, x := range values {
		if x == value {
			return true
		}
	}
	return false
}
