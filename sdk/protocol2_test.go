package sdk_test

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
	"testing"

	sdk_args "github.com/newrelic/infra-integrations-sdk/args"
	"github.com/newrelic/infra-integrations-sdk/metric"
	"github.com/newrelic/infra-integrations-sdk/sdk"
)

func TestNewEntityData(t *testing.T) {
	e, err := sdk.NewEntityData("TestEntityName", "TestEntityType")
	if err != nil {
		t.Error(err)
	}

	if e.Entity.Name != "TestEntityName" || e.Entity.Type != "TestEntityType" {
		t.Error("entity malformed")
	}
}

func TestNewEntityData_MissingData(t *testing.T) {
	e, err := sdk.NewEntityData("", "test")
	if err == nil {
		t.Error("error was expected on partial entity data")
	}

	emptyEntity := sdk.Entity{}
	if e.Entity != emptyEntity {
		t.Error("no entity expected")
	}

	e, err = sdk.NewEntityData("Entity", "")
	if err == nil {
		t.Error("error was expected on partial entity data")
	}

	if e.Entity != emptyEntity {
		t.Error("no entity expected")
	}

	e, err = sdk.NewEntityData("", "")
	if err != nil {
		t.Error(err)
	}

	if e.Entity != emptyEntity {
		t.Error("no entity expected")
	}
}

func TestNewIntegrationProtocol2Data(t *testing.T) {
	i, err := sdk.NewIntegrationProtocol2("TestIntegration", "1.0", new(struct{}))
	if err != nil {
		t.Fatal()
	}

	if i.Name != "TestIntegration" {
		t.Error()
	}
	if i.IntegrationVersion != "1.0" {
		t.Error()
	}
	if i.ProtocolVersion != "2" {
		t.Error()
	}
	if len(i.Data) != 0 {
		t.Error()
	}
}

func TestNewIntegrationProtocol2WithDefaultArguments(t *testing.T) {
	type argumentList struct {
		sdk_args.DefaultArgumentList
	}

	// Needed for initialising os.Args + flags (emulating).
	os.Args = []string{"cmd"}
	flag.CommandLine = flag.NewFlagSet("name", 0)

	var al argumentList
	i, err := sdk.NewIntegrationProtocol2("TestIntegration", "1.0", &al)
	if err != nil {
		t.Fatal(err)
	}
	if i.Name != "TestIntegration" {
		t.Error()
	}
	if i.IntegrationVersion != "1.0" {
		t.Error()
	}
	if i.ProtocolVersion != "2" {
		t.Error()
	}
	if len(i.Data) != 0 {
		t.Error()
	}
	if !al.All {
		t.Error()
	}
	if al.Pretty {
		t.Error()
	}
	if al.Verbose {
		t.Error()
	}
}

func TestIntegrationProtocol2_Publish(t *testing.T) {
	w := testWritter{
		func(p []byte) {
			expectedOutputRaw := []byte(`{"name":"TestIntegration","protocol_version":"2","integration_version":"1.0","data":[{"entity":{"name":"EntityOne","type":"test"},"metrics":[{"event_type":"EventTypeForEntityOne","entityName":"test:EntityOne","metricOne":99,"metricThree":"test","metricTwo":88}],"inventory":{},"events":[]},{"entity":{"name":"EntityTwo","type":"test"},"metrics":[{"event_type":"EventTypeForEntityTwo","entityName":"test:EntityTwo","metricOne":99,"metricThree":"test","metricTwo":88}],"inventory":{},"events":[]},{"entity":{},"metrics":[{"event_type":"EventTypeForEntityThree","metricOne":99,"metricThree":"test","metricTwo":88}],"inventory":{},"events":[]}]}`)
			expectedOutput := new(sdk.IntegrationProtocol2)
			err := json.Unmarshal(expectedOutputRaw, expectedOutput)
			if err != nil {
				t.Fatal("error unmarshaling expected output raw test data sample")
			}

			integration := new(sdk.IntegrationProtocol2)
			err = json.Unmarshal(p, integration)
			if err != nil {
				t.Error("error unmarshaling integration output", err)
			}

			if !reflect.DeepEqual(expectedOutput, integration) {
				t.Errorf("output does not match the expectations.\nGot:\n%v\nExpected:\n%v", p, expectedOutput)
			}
		},
	}

	i, err := sdk.NewIntegrationProtocol2WithWriter("TestIntegration", "1.0", new(struct{}), w)
	if err != nil {
		t.Fatal(err)
	}

	e, err := i.Entity("EntityOne", "test")
	if err != nil {
		t.Fatal(err)
	}

	ms := e.NewMetricSet("EventTypeForEntityOne")
	ms.SetMetric("metricOne", 99, metric.GAUGE)
	ms.SetMetric("metricTwo", 88, metric.GAUGE)
	ms.SetMetric("metricThree", "test", metric.ATTRIBUTE)

	e, err = i.Entity("EntityTwo", "test")
	if err != nil {
		t.Fatal(err)
	}

	ms = e.NewMetricSet("EventTypeForEntityTwo")
	ms.SetMetric("metricOne", 99, metric.GAUGE)
	ms.SetMetric("metricTwo", 88, metric.GAUGE)
	ms.SetMetric("metricThree", "test", metric.ATTRIBUTE)

	e, err = i.Entity("", "")
	if err != nil {
		t.Fatal(err)
	}

	ms = e.NewMetricSet("EventTypeForEntityThree")
	ms.SetMetric("metricOne", 99, metric.GAUGE)
	ms.SetMetric("metricTwo", 88, metric.GAUGE)
	ms.SetMetric("metricThree", "test", metric.ATTRIBUTE)

	i.Publish()
}

func TestIntegrationProtocol2_EntityReturnsExistingEntity(t *testing.T) {
	i, err := sdk.NewIntegrationProtocol2("TestIntegration", "1.0", new(struct{}))
	if err != nil {
		t.Fatal(err)
	}

	e1, err := i.Entity("Entity1", "test")
	if err != nil {
		t.Fail()
	}

	e2, err := i.Entity("Entity1", "test")
	if err != nil {
		t.Fail()
	}

	if e1 != e2 {
		t.Error("entity should be equal.")
	}

	if len(i.Data) > 1 {
		t.Error()
	}
}

// NOTE: This test does nothing as test but when running with -race flag we can detect data races.
// See Lock and Unlock on Entity method.
func TestIntegrationProtocol2_EntityHasNoDataRace(t *testing.T) {
	in, err := sdk.NewIntegrationProtocol2("TestIntegration", "1.0", new(struct{}))
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		go func(i int) {
			in.Entity(fmt.Sprintf("entity%v", i), "test")
		}(i)
	}
}

func TestAddNotificationEvent_Entity(t *testing.T) {
	en, err := sdk.NewEntityData("Entity1", "Type1")
	if err != nil {
		t.Fatal(err)
	}

	err = en.AddNotificationEvent("TestSummary")
	if err != nil {
		t.Errorf("error not expected, got: %s", err)
	}

	if en.Events[0].Summary != "TestSummary" || en.Events[0].Category != "notifications" {
		t.Error("event malformed")
	}

	if len(en.Events) != 1 {
		t.Error("not expected length of events")
	}
}

func TestAddNotificationEvent_Event_NoSummary_Error(t *testing.T) {
	en, err := sdk.NewEntityData("Entity1", "Type1")
	if err != nil {
		t.Fatal(err)
	}

	err = en.AddNotificationEvent("")
	if err == nil {
		t.Error("error was expected for empty summary")
	}

	if len(en.Events) != 0 {
		t.Error("not expected length of events")
	}
}

func TestAddEvent_Entity(t *testing.T) {
	en, err := sdk.NewEntityData("Entity1", "Type1")
	if err != nil {
		t.Fatal(err)
	}

	err = en.AddEvent(sdk.Event{Summary: "TestSummary", Category: "TestCategory"})
	if err != nil {
		t.Errorf("error not expected, got: %s", err)
	}

	if en.Events[0].Summary != "TestSummary" || en.Events[0].Category != "TestCategory" {
		t.Error("event malformed")
	}

	if len(en.Events) != 1 {
		t.Error("not expected length of events")
	}
}

func TestAddEvent_Entity_TheSameEvents_And_NoCategory(t *testing.T) {
	en, err := sdk.NewEntityData("Entity1", "Type1")
	if err != nil {
		t.Fatal(err)
	}

	err = en.AddEvent(sdk.Event{Summary: "TestSummary"})
	if err != nil {
		t.Errorf("error not expected, got: %s", err)
	}
	err = en.AddEvent(sdk.Event{Summary: "TestSummary"})
	if err != nil {
		t.Errorf("error not expected, got: %s", err)
	}

	if en.Events[0].Summary != "TestSummary" || en.Events[0].Category != "" {
		t.Error("event malformed")
	}
	if en.Events[1].Summary != "TestSummary" || en.Events[1].Category != "" {
		t.Error("event malformed")
	}
	if len(en.Events) != 2 {
		t.Error("not expected length of events")
	}
}

func TestAddEvent_Entity_EmptySummary_Error(t *testing.T) {
	en, err := sdk.NewEntityData("Entity1", "Type1")
	if err != nil {
		t.Fatal(err)
	}

	err = en.AddEvent(sdk.Event{Category: "TestCategory"})
	if err == nil {
		t.Error("error was expected for empty summary")
	}

	if len(en.Events) != 0 {
		t.Error("not expected length of events")
	}
}

type testWritter struct {
	testFunc func([]byte)
}

func (w testWritter) Write(p []byte) (n int, err error) {
	w.testFunc(p)

	return len(p), err
}