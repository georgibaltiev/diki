// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/diki/pkg/internal/utils"
)

var _ = Describe("utils", func() {
	DescribeTable("#EqualSets",
		func(s1, s2 []string, expectedResult bool) {
			result := utils.EqualSets(s1, s2)

			Expect(result).To(Equal(expectedResult))
		},
		Entry("should return true when s1 and s2 have same elements ordered",
			[]string{"foo", "bar"}, []string{"foo", "bar"}, true),
		Entry("should return true when s1 and s2 have same elements not ordered",
			[]string{"bar", "foo"}, []string{"foo", "bar"}, true),
		Entry("should return false when s1 and s2 have different elements",
			[]string{"foo", "bar"}, []string{"foo", "bar", "foo-bar"}, false),
	)

	DescribeTable("#Subset",
		func(s1, s2 []string, expectedResult bool) {
			result := utils.Subset(s1, s2)

			Expect(result).To(Equal(expectedResult))
		},
		Entry("should return true when s1 is empty",
			[]string{}, []string{"foo", "bar"}, true),
		Entry("should return true when s1 is a subset of s2",
			[]string{"bar", "foo"}, []string{"foo", "bar", "foo-bar"}, true),
		Entry("should return false when s1 is not a subset of s2",
			[]string{"foo", "foo-bar"}, []string{"foo", "bar", "test"}, false),
		Entry("should return false when s1 has more elements than s2",
			[]string{"foo", "bar", "foo-bar"}, []string{"foo", "bar"}, false),
	)

	DescribeTable("#Intersect",
		func(s1, s2 []string, expectedResult bool) {
			result := utils.Intersect(s1, s2)

			Expect(result).To(Equal(expectedResult))
		},
		Entry("should return false when s1 is empty",
			[]string{}, []string{"foo", "bar"}, false),
		Entry("should return false when both s1 and s2 are empty",
			[]string{}, []string{}, false),
		Entry("should return true when s1 and s2 intersect",
			[]string{"bar", "foo", "baz"}, []string{"foo", "bar", "foo-bar"}, true),
		Entry("should return false when s1 and s2 do not intersect",
			[]string{"baz", "foo-bar", "qux"}, []string{"foo", "bar", "test"}, false),
	)

	DescribeTable("#InitialSegment",
		func(s1, s2 []string, expectedResult bool) {
			result := utils.StartsWith(s1, s2...)

			Expect(result).To(Equal(expectedResult))
		},
		Entry("should return true when s1 is empty",
			[]string{"foo", "bar"}, []string{}, true),
		Entry("should return true when s1 is a initial segment of s2",
			[]string{"foo", "bar", "foo-bar"}, []string{"foo", "bar"}, true),
		Entry("should return false when s1 is not a initial segment of s2",
			[]string{"foo", "bar", "test"}, []string{"bar", "test"}, false),
		Entry("should return false when s1 has more elements than s2",
			[]string{"foo", "bar"}, []string{"foo", "bar", "foo-bar"}, false),
	)

	DescribeTable("#MatchLabels",
		func(m1, m2 map[string]string, expectedResult bool) {
			result := utils.MatchLabels(m1, m2)

			Expect(result).To(Equal(expectedResult))
		},
		Entry("should return true when m1 contains all keys and values of m2",
			map[string]string{"foo": "bar", "key1": "value1", "key2": "value2"},
			map[string]string{"key1": "value1", "key2": "value2"}, true),
		Entry("should return false when m1 does not contain all keys and values of m2",
			map[string]string{"key1": "value1", "key2": "value2"},
			map[string]string{"key1": "value1", "foo": "bar"}, false),
		Entry("should return false when m1 is nil",
			nil, map[string]string{"key1": "value1", "foo": "bar"}, false),
		Entry("should return false when m2 is nil",
			map[string]string{"key1": "value1", "foo": "bar"}, nil, false),
	)
})
