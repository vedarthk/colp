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
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"
)

var DryRun bool
var PidFile string

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
		os.Exit(1)
	}
	processToMonitor := args[0]
	log.Info("there are ", len(pids), " processes running")

	if PidFile != "" {
		bytes, err := ioutil.ReadFile(PidFile)
		if err != nil {
			log.Error("cannot read file: ", err.Error())
			os.Exit(1)
		}
		pid, err := strconv.ParseInt(strings.Trim(string(bytes), "\n"), 10, 32)

		if err != nil {
			log.Error("cannot parse PID from file ", err.Error())
			os.Exit(1)
		}
		log.Info("looking for PID=", pid)
		if contains(pids, int32(pid)) {
			log.Info(processToMonitor, " is running")
			pushMetric(processToMonitor, 1)
		} else {
			log.Info("PID=", pid, " not found")
			log.Info("found these PIDs=", pids)
			log.Error(processToMonitor, " is not running")
		}
	} else {
		procNames := []string{}
		for _, pid := range pids {
			proc := process.Process{Pid: pid}
			procName, err := proc.Name()
			if err != nil {
				log.Error("error getting name", err.Error())
			}
			procNames = append(procNames, procName)
		}
		if len(intersect(procNames, []string{processToMonitor})) > 0 {
			log.Info(processToMonitor, " is running")
			pushMetric(processToMonitor, 1)
		} else {
			log.Error(processToMonitor, " is not running")
		}
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
	pushCmd.Flags().StringVarP(&PidFile, "pidfile", "f", "", "PID file")
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

func contains(values interface{}, value interface{}) bool {
	tempArr := reflect.ValueOf(values)
	length := tempArr.Len()

	for i := 0; i < length; i++ {
		switch t := value.(type) {
		case int32:
			if tempArr.Index(i).Interface().(int32) == value.(int32) {
				return true
			}
		case string:
			if tempArr.Index(i).Interface().(string) == value.(string) {
				return true
			}
		default:
			log.Errorf("%T type not handled", t)
		}
	}
	return false
}
