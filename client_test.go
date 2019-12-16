package redis_timeseries_go

import (
	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
	"os"
	"reflect"
	"testing"
	"time"
)

func createClient() *Client {
	valueh, exists := os.LookupEnv("REDISTIMESERIES_TEST_HOST")
	host := "localhost:6379"
	if exists && valueh != "" {
		host = valueh
	}
	valuep, exists := os.LookupEnv("REDISTIMESERIES_TEST_PASSWORD")
	password := "SUPERSECRET"
	var ptr *string = nil
	if exists {
		password = valuep
	}
	if len(password) > 0 {
		ptr = MakeStringPtr(password)
	}
	return NewClient(host, "test_client", ptr)
}

var defaultDuration, _ = time.ParseDuration("1h")
var tooShortDuration, _ = time.ParseDuration("10ms")

func (client *Client) FlushAll() (err error) {
	conn := client.Pool.Get()
	defer conn.Close()
	_, err = conn.Do("FLUSHALL")
	return err
}

func init() {
	client := createClient()
	client.FlushAll()
}

func TestCreateKey(t *testing.T) {
	client := createClient()
	err := client.CreateKey("test_CreateKey", defaultDuration)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	labels := map[string]string{
		"cpu":     "cpu1",
		"country": "IT",
	}
	_, err = client.CreateKeyWithOptions("test_CreateKeyLabels", CreateOptions{RetentionSecs: defaultDuration, Labels: labels})
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	err = client.CreateKey("test_CreateKey", tooShortDuration)
	assert.NotNil(t, err)
}

func TestCreateRule(t *testing.T) {
	client := createClient()
	var destinationKey string
	var err error
	key := "test_CreateRule"
	err = client.CreateKey(key, defaultDuration)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	var found bool
	for aggType, aggString := range aggToString {
		destinationKey = "test_CreateRule_dest" + aggString
		err = client.CreateKey(destinationKey, defaultDuration)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		err = client.CreateRule(key, aggType, 100, destinationKey)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		info, err := client.Info(key)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		found = false
		for _, rule := range info.Rules {
			if aggType == rule.AggType {
				found = true
			}
		}
		assert.True(t, found)
	}
}

func TestClientInfo(t *testing.T) {
	client := createClient()
	key := "test_INFO"
	destKey := "test_INFO_dest"
	err := client.CreateKey(key, defaultDuration)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	err = client.CreateKey(destKey, defaultDuration)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	err = client.CreateRule(key, AvgAggregation, 100, destKey)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	res, err := client.Info(key)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	expected := KeyInfo{ChunkCount: 1,
		MaxSamplesPerChunk: 256, LastTimestamp: 0, RetentionTime: 3600000,
		Rules: []Rule{{DestKey: destKey, BucketSizeSec: 100, AggType: AvgAggregation},
		},
		Labels: map[string]string{},
	}
	assert.Equal(t, expected, res)
}

func TestDeleteRule(t *testing.T) {
	client := createClient()
	key := "test_DELETE"
	destKey := "test_DELETE_dest"
	err := client.CreateKey(key, defaultDuration)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	err = client.CreateKey(destKey, defaultDuration)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	err = client.CreateRule(key, AvgAggregation, 100, destKey)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	err = client.DeleteRule(key, destKey)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	assert.Equal(t, nil, err)
	info, err := client.Info(key)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	assert.Equal(t, 0, len(info.Rules))
	err = client.DeleteRule(key, destKey)
	assert.Equal(t, redis.Error("TSDB: compaction rule does not exist"), err)
}

func TestAdd(t *testing.T) {
	client := createClient()
	key := "test_ADD"
	now := time.Now().Unix()
	PI := 3.14159265359
	err := client.CreateKey(key, defaultDuration)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	storedTimestamp, err := client.Add(key, now, PI)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	assert.Equal(t, now, storedTimestamp)
	info, err := client.Info(key)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	assert.Equal(t, now, info.LastTimestamp)
}

func TestAddWithRetention(t *testing.T) {
	client := createClient()
	// There is no way I know of yet that allows me to query the retention for a single datapoint
	// this test should probably be improved
	key := "test_ADDWITHRETENTION"
	now := time.Now().Unix()
	PI := 3.14159265359
	err := client.CreateKey(key, defaultDuration)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	_, err = client.AddWithRetention(key, now, PI, 2112)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	info, err := client.Info(key)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	assert.Equal(t, now, info.LastTimestamp)
}

func TestClient_Range(t *testing.T) {
	client := createClient()
	key := "test_Range"
	err := client.CreateKey(key, defaultDuration)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	now := time.Now().Unix()
	pi := 3.14159265359
	halfPi := pi / 2

	_, err = client.Add(key, now-2, halfPi)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	_, err = client.Add(key, now, pi)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	dataPoints, err := client.Range(key, now-1, now)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	expected := []DataPoint{{Timestamp: now, Value: pi}}
	assert.Equal(t, expected, dataPoints)

	dataPoints, err = client.Range(key, now-2, now)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	expected = []DataPoint{{Timestamp: now - 2, Value: halfPi}, {Timestamp: now, Value: pi}}
	assert.Equal(t, expected, dataPoints)

	dataPoints, err = client.Range(key, now-4, now-3)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	expected = []DataPoint{}
	assert.Equal(t, expected, dataPoints)

	_, err = client.Range(key+"1", now-1, now)
	assert.NotNil(t, err)
}

func TestClient_AggRange(t *testing.T) {
	client := createClient()
	key := "test_aggRange"
	err := client.CreateKey(key, defaultDuration)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	now := int64(1552839965)
	value := 5.0
	value2 := 6.0

	_, err = client.Add(key, now-2, value)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	_, err = client.Add(key, now-1, value2)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	dataPoints, err := client.AggRange(key, now-60, now, CountAggregation, 10)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	assert.Equal(t, 2.0, dataPoints[0].Value)

	_, err = client.AggRange(key+"1", now-60, now, CountAggregation, 10)
	assert.NotNil(t, err)
}

func TestClient_AggMultiRange(t *testing.T) {
	client := createClient()
	key := "test_aggMultiRange1"
	labels := map[string]string{
		"cpu":     "cpu1",
		"country": "US",
	}
	now := int64(1552839965)
	_, err := client.AddWithOptions(key, now-2, 5.0, CreateOptions{RetentionSecs: defaultDuration, Labels: labels})
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	_, err = client.AddWithOptions(key, now-1, 6.0, CreateOptions{RetentionSecs: defaultDuration, Labels: labels})
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	key2 := "test_aggMultiRange2"
	labels2 := map[string]string{
		"cpu":     "cpu2",
		"country": "US",
	}
	_, err = client.CreateKeyWithOptions(key2, CreateOptions{RetentionSecs: defaultDuration, Labels: labels2})
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	_, err = client.AddWithOptions(key2, now-2, 4.0, CreateOptions{})
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	_, err = client.Add(key2, now-1, 8.0)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	ranges, err := client.AggMultiRange(now-60, now, CountAggregation, 10, "country=US")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	assert.Equal(t, 2, len(ranges))
	assert.Equal(t, 2.0, ranges[0].DataPoints[0].Value)

	_, err = client.AggMultiRange(now-60, now, CountAggregation, 10)
	assert.NotNil(t, err)

}

func TestParseLabels(t *testing.T) {
	type args struct {
		res interface{}
	}
	tests := []struct {
		name       string
		args       args
		wantLabels map[string]string
		wantErr    bool
	}{
		{"correctInput",
			args{[]interface{}{[]interface{}{[]byte("hostname"), []byte("host_3")}, []interface{}{[]byte("region"), []byte("us-west-2")}}},
			map[string]string{"hostname": "host_3", "region": "us-west-2",},
			false,
		},
		{"IncorrectInput",
			args{[]interface{}{[]interface{}{[]byte("hostname"), []byte("host_3")}, []interface{}{[]byte("region"),}}},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLabels, err := ParseLabels(tt.args.res)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseLabels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotLabels, tt.wantLabels) {
				t.Errorf("ParseLabels() gotLabels = %v, want %v", gotLabels, tt.wantLabels)
			}
		})
	}
}
