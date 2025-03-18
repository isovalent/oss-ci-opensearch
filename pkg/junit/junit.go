package junit

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/jstemmer/go-junit-report/v2/junit"

	"github.com/isovalent/corgi/pkg/types"
	"github.com/isovalent/corgi/pkg/util"
)

func parseTestsuite(
	suite *junit.Testsuite,
	run *types.WorkflowRun,
	allowedTestConclusions []string,
	l *slog.Logger,
) (*types.Testsuite, []types.Testcase, error) {
	s := &types.Testsuite{
		WorkflowRun:   run,
		Type:          types.TypeNameTestsuite,
		Name:          suite.Name,
		TotalTests:    suite.Tests,
		TotalFailures: suite.Failures,
		TotalErrors:   suite.Errors,
		TotalSkipped:  suite.Skipped,
	}

	if suite.Time != "" {
		duration, err := time.ParseDuration(fmt.Sprintf("%ss", suite.Time))
		if err != nil {
			return nil, nil, fmt.Errorf("unable to parse duration '%ss': %w", suite.Time, err)
		}
		s.Duration = duration
	}

	if suite.Timestamp != "" {
		// ISO8601.
		// Some timestamps have a "Z" at the end, and some don't.
		// The time package complains if the given time to parse doesn't exactly
		// match the given format, therefore we need to trim the Z if it's in the timestamp.
		endTime, err := time.Parse("2006-01-02T15:04:05", strings.TrimSuffix(suite.Timestamp, "Z"))
		if err != nil {
			return nil, nil, fmt.Errorf("unable to parse timestamp '%s': %w", suite.Timestamp, err)
		}
		s.EndTime = endTime
	}

	cases := []types.Testcase{}

	for _, testcase := range suite.Testcases {
		tc := types.Testcase{
			Testsuite: s,
			Type:      types.TypeNameTestcase,
			Name:      testcase.Name,
		}

		// There are a couple of formats for the cilium-junits. Sometimes
		// the Status property is set, and other times it isn't. It if isn't set,
		// the status will be exposed through the different
		// result fields of the junit.Testcase.

		if testcase.Status != "" {
			tc.Status = testcase.Status
		} else {
			if testcase.Error != nil {
				tc.Status = "error"
			} else if testcase.Failure != nil {
				tc.Status = "failure"
			} else if testcase.Skipped != nil {
				tc.Status = "skipped"
			} else {
				tc.Status = "passed"
			}
		}

		if !util.Contains(allowedTestConclusions, tc.Status) {
			l.Debug(
				"Skipping test case for workflow, does not meet status criteria",
				"testcase-name", testcase.Name, "testcase-status", testcase.Status,
			)

			continue
		}

		if testcase.Time != "" {
			duration, err := time.ParseDuration(fmt.Sprintf("%ss", testcase.Time))
			if err != nil {
				return nil, nil, fmt.Errorf("unable to parse duration '%ss': %w", testcase.Time, err)
			}
			tc.Duration = duration
		}

		cases = append(cases, tc)
	}

	return s, cases, nil
}

func ParseFiles(
	files []*zip.File,
	run *types.WorkflowRun,
	allowedTestConclusions []string,
	l *slog.Logger,
) ([]types.Testsuite, []types.Testcase, error) {
	suites := []types.Testsuite{}
	cases := []types.Testcase{}

	for _, fil := range files {
		if !strings.HasSuffix(fil.Name, ".xml") || fil.FileInfo().IsDir() {
			l.Debug("ignoring non-xml file in cilium-junits archive", "file", fil.Name)
			continue
		}

		l.Info("Parsing JUnit file", "name", fil.Name)

		fileReader, err := fil.Open()
		if err != nil {
			return nil, nil, fmt.Errorf("unable to open file %q: %w", fil.Name, err)
		}
		defer fileReader.Close()

		buf := &bytes.Buffer{}

		_, err = io.Copy(buf, fileReader)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to read junit file %q: %w", fil.Name, err)
		}

		// Sometimes a JUnit file can be empty, so we need to rule out empty files.
		if buf.Len() == 0 {
			continue
		}

		// A JUnit file may either be:
		// 1. A junit.Testsuites object with multiple junit.Testsuite objects.
		// 2. A junit.Testsuites object with a single junit.Testsuite object.
		// 3. A single junit.Testsuite.
		// Try all options when unmarshalling.
		// Note that the XML parser thinks the Testsuites object is a valid Testsuite object, so
		// we have to try parsing into a Testsuites first.
		toParse := []junit.Testsuite{}
		s := junit.Testsuites{}
		if err := xml.Unmarshal(buf.Bytes(), &s); err != nil {
			s := junit.Testsuite{}
			if err2 := xml.Unmarshal(buf.Bytes(), &s); err2 != nil {
				e := errors.Join(err, err2)
				return nil, nil, fmt.Errorf("unable to unmarshal junit file '%s' in artifact to Testsuite or Testsuites object: %w", fil.Name, e)
			}
			toParse = append(toParse, s)
		} else {
			toParse = s.Suites
		}

		for _, s := range toParse {
			parsedSuite, parsedCases, err := parseTestsuite(&s, run, allowedTestConclusions, l)
			if err != nil {
				return nil, nil, fmt.Errorf("unable to parse test suite in junit file '%s': %w", fil.Name, err)
			}

			parsedSuite.JUnitFilename = fil.Name
			suites = append(suites, *parsedSuite)
			cases = append(cases, parsedCases...)
		}

	}

	return suites, cases, nil
}
