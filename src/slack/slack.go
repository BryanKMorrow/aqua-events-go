package slack

import (
	"encoding/json"
	"fmt"
	"github.com/BryanKMorrow/aqua-events-go/src/aqua"
	"github.com/slack-go/slack"
	"log"
	"strconv"
	"strings"
	"time"
)

const (
	// AuthorName is the message identifier
	AuthorName = "aqua-events"
	// Fallback is the backup for AuthorName
	Fallback = "Aqua Security Audit Events"
	// AuthorSubname follows the AuthorName in the header
	AuthorSubname = "AquaEvents"
	// AuthorLink points to the github repo for this application
	AuthorLink = "https://github.com/BryanKMorrow/aqua-events-go"
	// AuthorIcon points to the Aqua favicon
	AuthorIcon = "https://www.aquasec.com/wp-content/themes/aqua3/favicon.ico"
)

// Message is the slack struct
type Message struct {
	Attachment slack.Attachment `json:"attachment"`
	Webhook    string           `json:"webhook"`
	IgnoreList []string         `json:"ignore_list"`
}

// ProcessAudit receives the post data and sends to slack
func (m *Message) ProcessAudit(audit aqua.Audit) {
	// format the message
	ignore := false
	msg := m.Format(audit)

	if audit.Result == 2 { // BLOCK
		contains := sliceContains(m.IgnoreList, "block")
		if contains {
			log.Println("ignoring block events")
			ignore = true
		}
	} else if audit.Result == 1 { // SUCCESS
		contains := sliceContains(m.IgnoreList, "success")
		if contains {
			log.Println("ignoring success events")
			ignore = true
		}
	} else if audit.Result == 3 { // DETECT
		contains := sliceContains(m.IgnoreList, "detect")
		if contains {
			log.Println("ignoring detect events")
			ignore = true
		}
	} else if audit.Result == 4 {
		contains := sliceContains(m.IgnoreList, "alert")
		if contains {
			log.Println("ignoring critical events")
			ignore = true
		}
	}
	if !ignore {
		err := slack.PostWebhook(m.Webhook, &msg)
		if err != nil {
			log.Println("failed posting attachment to Slack API: %w", err)
		}
	}
}

func (m *Message) Format(audit aqua.Audit) slack.WebhookMessage {
	var text string
	var err error
	var a []byte
	// base attachment settings
	m.Attachment.Fallback = Fallback
	m.Attachment.AuthorName = AuthorName
	m.Attachment.AuthorSubname = AuthorSubname
	m.Attachment.AuthorIcon = AuthorIcon
	// format based on message level
	if audit.Result == 1 {
		m.Attachment.Color = "good"
		if audit.Type == "Administration" {
			text = fmt.Sprintf("Type: %s\nAction: %s\nPerformed On: %s\nPerformed By: %s\nAqua Response: %s\nTimestamp: %s\n",
				audit.Type, audit.Action, fmt.Sprintf("%s %s", audit.Category, audit.Adjective), audit.User, "Success", time.Unix(int64(audit.Time), 0).Format(time.RFC822Z))
			m.Attachment.AuthorSubname = fmt.Sprintf("User %s performed %s on %s", audit.User, audit.Action, fmt.Sprintf("%s %s", audit.Category, audit.Adjective))
		} else if audit.Type == "CVE" || audit.Category == "CVE" {
			log.Println("Data: ", audit.Data)
			text = fmt.Sprintf("Image: %s\nImage Hash: %s\nRegistry: %s\nImage added by user: %s\nImage scan start time: %s\nImage scan end time: %s\nAqua Response: %s\nTimestamp: %s\n",
				audit.Image, audit.Imagehash, audit.Registry, audit.User, time.Unix(int64(audit.Time), 0).Format(time.RFC822Z), time.Unix(int64(audit.Time), 0).Format(time.RFC822Z),
				"Success", time.Unix(int64(audit.Time), 0).Format(time.RFC822Z))
			m.Attachment.AuthorSubname = fmt.Sprintf("Scan of image %s from registry %s revealed no security issues", audit.Image, audit.Registry)
		} else if audit.Type == "Docker" || audit.Category == "container" || audit.Category == "image" {
			text = fmt.Sprintf("Host: %s\nHost IP: %s\nImage Name: %s\nContainer Name: %s\nAction: %s\nKubernetes Cluster: %s\nVM Location: %s\nAqua Response: %s\nAqua Policy: %s\nDetails: %s\n"+
				"Enforcer Group: %s\nTime Stamp: %s\n", audit.Host, audit.Hostip, audit.Image, audit.Container, audit.Action, audit.K8SCluster, audit.VMLocation, "Success", audit.Rule, audit.Reason,
				audit.Hostgroup, time.Unix(int64(audit.Time), 0).Format(time.RFC822Z))
			m.Attachment.AuthorSubname = fmt.Sprintf("User ran action %s on host %s", audit.Action, audit.Host)
		} else {
			a, err = json.Marshal(&audit)
			if err != nil {
				log.Println(err)
			}
			text = string(a)
		}
	} else if audit.Result == 3 {
		m.Attachment.Color = "warning"
		if audit.Category == "CVE" {
			text = fmt.Sprintf("Image: %s\nRegistry: %s\nImage was addded by user %s\nImage scan start time: %s\nImage scan end time: %s\n"+
				"Aqua Response: %s\nTime Stamp: %s\n", audit.Image, audit.Registry, audit.User,
				time.Unix(int64(audit.Time), 0).Format(time.RFC822Z), time.Unix(int64(audit.Time), 0).Format(time.RFC822Z), "Detect", time.Unix(int64(audit.Time), 0).Format(time.RFC822Z))
			m.Attachment.AuthorSubname = fmt.Sprintf("Scan of image %s from registry %s revealed %d total vulnerabilities",
				audit.Image, audit.Registry, audit.Critical+audit.High+audit.Medium+audit.Low)
		} else if audit.Category == "container" || audit.Category == "file" || audit.Category == "secret" {
			text = fmt.Sprintf("Host: %s\nHost IP: %s\nImage Name: %s\nContainer Name: %s\nAction: %s\nKubernetes Cluster: %s\nPod Name: %s\nPod Namespace: %s\nVM Location: %s\nAqua Response: %s\nAqua Policy: %s\nResource: %s\nCommand: %s\nSecurity Control: %s\n"+
				"Details: %s\nEnforcer Group: %s\nTime Stamp: %s\n", audit.Host, audit.Hostip, audit.Image, audit.Container, audit.Action, audit.K8SCluster, audit.Podname, audit.Podnamespace, audit.VMLocation, "Detect", audit.Rule, audit.Resource, audit.Command, audit.Control,
				audit.Reason, audit.Hostgroup, time.Unix(int64(audit.Time), 0).Format(time.RFC822Z))
			m.Attachment.AuthorSubname = fmt.Sprintf("User ran command %s on host %s", audit.Action, audit.Host)
		} else {
			a, err = json.Marshal(&audit)
			if err != nil {
				log.Println(err)
			}
			text = string(a)
		}
	} else if audit.Result == 2 {
		m.Attachment.Color = "danger"
		if audit.Category == "container" || audit.Category == "file" || audit.Category == "secret" {
			text = fmt.Sprintf("Host: %s\nHost IP: %s\nImage Name: %s\nContainer Name: %s\nAction: %s\nKubernetes Cluster: %s\nPod Name: %s\nPod Namespace: %s\nVM Location: %s\nAqua Response: %s\nAqua Policy: %s\nResource: %s\nCommand: %s\nSecurity Control: %s\n"+
				"Details: %s\nEnforcer Group: %s\nTime Stamp: %s\n", audit.Host, audit.Hostip, audit.Image, audit.Container, audit.Action, audit.K8SCluster, audit.Podname, audit.Podnamespace, audit.VMLocation, "Detect", audit.Rule, audit.Resource, audit.Command, audit.Control,
				audit.Reason, audit.Hostgroup, time.Unix(int64(audit.Time), 0).Format(time.RFC822Z))
			m.Attachment.AuthorSubname = fmt.Sprintf("User ran command %s on host %s", audit.Action, audit.Host)
		} else {
			a, err = json.Marshal(&audit)
			if err != nil {
				log.Println(err)
			}
			text = string(a)
		}
	} else if audit.Result == 4 {
		var data aqua.Data
		err = json.Unmarshal([]byte(audit.Data), &data)
		if err != nil {
			log.Println("error while unmarshalling alert data field ", err)
		}
		var control string
		m.Attachment.Color = "danger"
		if audit.Category == "image" {
			if data.Blocking && data.Pending {
				control = "non-compliant container(s) already running"
			} else {
				control = "non-compliant"
			}
			var data aqua.Data
			err = json.Unmarshal([]byte(audit.Data), &data)
			if err != nil {
				log.Println("error while unmarshalling alert data field ", err)
				splitBadData(audit.Data)

			}
			var controls string
			for i, str := range data.Controls {
				if i == 0 {
					controls = str
				} else {
					controls = controls + ", " + str
				}
			}
			strings.TrimSuffix(controls, ",")
			text = fmt.Sprintf("Entity: %s\nImage: %s\nAction taken: %s\nPolicy: %s\nFailed Controls: %s\nRegistry: %s\n Aqua Response: %s\nTime Stamp: %s",
				"Image", audit.Image, audit.Action, data.PolicyName, controls, data.Registry, "Alert", time.Unix(int64(audit.Time), 0).Format(time.RFC822Z))
			m.Attachment.AuthorSubname = fmt.Sprintf("Image %s is %s", audit.Image, control)
		} else {
			a, err = json.Marshal(&audit)
			if err != nil {
				log.Println(err)
			}
			text = string(a)
		}
	}
	m.Attachment.Text = text
	m.Attachment.Ts = json.Number(strconv.FormatInt(time.Now().Unix(), 10))
	msg := slack.WebhookMessage{
		Attachments: []slack.Attachment{m.Attachment},
	}
	return msg
}

// sliceContains checks for a string in a slice
func sliceContains(s []string, str string) bool {
	for _, v := range s {
		if v == strings.TrimSpace(str) {
			return true
		}
	}
	return false
}

func splitBadData(d string) {
	splits := strings.Split(d, ",")
	log.Println("SPLITS: ", splits)
	for _, str := range splits {
		cSplit := strings.Split(str, ":")
		log.Println("CSPLIT: ", cSplit)
		for _, c := range cSplit {
			log.Println("Split key and value: ", c)
		}
	}
}
