// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/gardener/diki/pkg/rule"
	"github.com/gardener/diki/pkg/ruleset"
)

type junitTestProperty struct {
	Message string `xml:"message,attr"`
	Text    string `xml:",chardata"`
}

type junitTestCase struct {
	Name       string                   `xml:"name,attr"`
	Failure    *junitTestProperty       `xml:"failure"`
	Error      *junitTestProperty       `xml:"error"`
	Skipped    *junitTestProperty       `xml:"skipped"`
	Properties *[]junitTestCaseProperty `xml:"properties"`
}

type junitTestSuite struct {
	Name      string          `xml:"name,attr"`
	TestCases []junitTestCase `xml:"testcase"`
}

type junitTestCaseProperty struct {
	Element junitPropertyElement `xml:"property"`
}

type junitPropertyElement struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type junitTestSuites struct {
	Name   string           `xml:"name,attr"`
	Suites []junitTestSuite `xml:"testsuite"`
}

func ParseTestNGReport(xmlContent string) (ruleset.RulesetResult, error) {

	var (
		testSuites junitTestSuites
		autoID     = 1
		makeID     = func() string {
			id := fmt.Sprintf("AUTO-%d", autoID)
			autoID++
			return id
		}
	)

	dec := xml.NewDecoder(strings.NewReader(xmlContent))
	if err := dec.Decode(&testSuites); err != nil {
		return ruleset.RulesetResult{}, err
	}

	if len(testSuites.Suites) == 0 && testSuites.Name == "" {
		return ruleset.RulesetResult{}, nil
	}

	rs := ruleset.RulesetResult{
		RuleResults: []rule.RuleResult{},
	}

	for _, suite := range testSuites.Suites {
		for _, tc := range suite.TestCases {
			var (
				ruleID   string
				ruleName = tc.Name
				status   = rule.Passed
				message  = ""
			)

			if tc.Properties != nil {
				for _, p := range *tc.Properties {
					if p.Element.Name == "security_id" {
						ruleID = p.Element.Value
					}
				}
			}

			if len(ruleID) == 0 {
				ruleID = makeID()
			}

			if tc.Failure != nil {
				status = rule.Failed
				if tc.Failure.Message != "" {
					message = tc.Failure.Message
				} else if tc.Failure.Text != "" {
					message = tc.Failure.Text
				}
			}

			if tc.Error != nil {
				status = rule.Errored
				if tc.Error.Message != "" {
					message = tc.Error.Message
				} else if tc.Error.Text != "" {
					message = tc.Error.Text
				}
			}

			if tc.Skipped != nil {
				status = rule.Skipped
				if tc.Skipped.Message != "" {
					message = tc.Skipped.Message
				} else if tc.Skipped.Text != "" {
					message = tc.Skipped.Text
				}
			}

			check := rule.CheckResult{
				Status: status,
			}

			if len(message) > 0 {
				check.Message = message
			}

			rs.RuleResults = append(rs.RuleResults, rule.RuleResult{
				RuleID:       ruleID,
				RuleName:     ruleName,
				CheckResults: []rule.CheckResult{check},
			})
		}
	}

	return rs, nil
}
