package base

import (
	"reflect"
	"sync/atomic"
	"testing"
	"unsafe"

	"github.com/alibaba/sentinel-golang/util"
	"github.com/stretchr/testify/mock"
)

const (
	// timespan of per slot
	BucketLengthInMs uint32 = 500
	// the number of slots
	SampleCount uint32 = 20
	// interval(ms) of sliding window, 10s
	IntervalInMs uint32 = 10 * 1000
)

func Test_bucketWrapper_Size(t *testing.T) {
	type Obj struct {
		a1 int32 // 4bytes
		a2 int32
		a3 int32
		a4 int32
		a5 int32
		a6 int32
		a7 int32
		a8 int32
	}
	ww := &bucketWrap{
		bucketStart: util.CurrentTimeMillis(),
		value:       atomic.Value{},
	}
	if unsafe.Sizeof(*ww) != 24 {
		t.Errorf("the size of bucketWrap is not equal 20.\n")
	}
	if unsafe.Sizeof(ww) != 8 {
		t.Errorf("the size of bucketWrap is not equal 20.\n")
	}
}

//type metricBucketMock struct {
//	mock.Mock
//}

// mock ArrayMock and implement bucketGenerator
type leapArrayMock struct {
	mock.Mock
}

func (bla *leapArrayMock) newEmptyBucket() interface{} {
	return new(int64)
}

func (bla *leapArrayMock) resetBucketTo(ww *bucketWrap, startTime uint64) *bucketWrap {
	ww.bucketStart = startTime
	ww.value.Store(new(int64))
	return ww
}

func Test_leapArray_calculateTimeIdx_normal(t *testing.T) {
	type fields struct {
		bucketLengthInMs uint32
		sampleCount      uint32
		intervalInMs     uint32
		array            *atomicBucketWrapArray
		mux              mutex
	}
	type args struct {
		timeMillis uint64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name: "Test_leapArray_calculateTimeIdx_normal",
			fields: fields{
				bucketLengthInMs: BucketLengthInMs,
				sampleCount:      SampleCount,
				intervalInMs:     IntervalInMs,
				array:            newAtomicBucketWrapArray(int(SampleCount), BucketLengthInMs, &leapArrayMock{}),
				mux:              mutex{},
			},
			args: args{
				timeMillis: 1576296044907,
			},
			want: 9,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			la := &leapArray{
				bucketLengthInMs: tt.fields.bucketLengthInMs,
				sampleCount:      tt.fields.sampleCount,
				intervalInMs:     tt.fields.intervalInMs,
				array:            tt.fields.array,
				updateLock:       tt.fields.mux,
			}
			if got := la.calculateTimeIdx(tt.args.timeMillis); got != tt.want {
				t.Errorf("leapArray.calculateTimeIdx() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_calculateStartTime_normal(t *testing.T) {
	type fields struct {
	}
	type args struct {
		timeMillis       uint64
		bucketLengthInMs uint32
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   uint64
	}{
		{
			name:   "Test_calculateStartTime_normal",
			fields: fields{},
			args: args{
				timeMillis:       1576296044907,
				bucketLengthInMs: BucketLengthInMs,
			},
			want: 1576296044500,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateStartTime(tt.args.timeMillis, tt.args.bucketLengthInMs); got != tt.want {
				t.Errorf("leapArray.calculateStartTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_leapArray_BucketStartCheck_normal(t *testing.T) {
	type fields struct {
		BucketLengthInMs uint32
		sampleCount      uint32
		intervalInMs     uint32
		array            *atomicBucketWrapArray
		mux              mutex
	}
	type args struct {
		bg         bucketGenerator
		timeMillis uint64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   uint64 //start time of bucket
	}{
		{
			name: "Test_leapArray_BucketStartCheck_normal",
			fields: fields{
				BucketLengthInMs: BucketLengthInMs,
				sampleCount:      SampleCount,
				intervalInMs:     IntervalInMs,
				array:            newAtomicBucketWrapArray(int(SampleCount), BucketLengthInMs, &leapArrayMock{}),
				mux:              mutex{},
			},
			args: args{
				bg:         new(leapArrayMock),
				timeMillis: 1576296044907,
			},
			want: 1576296044500,
		},
	}
	wwPtr := tests[0].fields.array.get(9)
	wwPtr.bucketStart = 1576296044500 //start time of cycle 1576296040000

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			la := &leapArray{
				bucketLengthInMs: tt.fields.BucketLengthInMs,
				sampleCount:      tt.fields.sampleCount,
				intervalInMs:     tt.fields.intervalInMs,
				array:            tt.fields.array,
				updateLock:       tt.fields.mux,
			}
			got, err := la.currentBucketOfTime(tt.args.timeMillis, tt.args.bg)
			if err != nil {
				t.Errorf("leapArray.currentBucketOfTime() error = %v\n", err)
				return
			}
			if got.bucketStart != tt.want {
				t.Errorf("bucketStart = %v, want %v", got.bucketStart, tt.want)
			}
		})
	}
}

func Test_leapArray_currentBucketWithTime_normal(t *testing.T) {
	type fields struct {
		bucketLengthInMs uint32
		sampleCount      uint32
		intervalInMs     uint32
		array            *atomicBucketWrapArray
		mux              mutex
	}
	type args struct {
		bg         bucketGenerator
		timeMillis uint64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *bucketWrap
		wantErr bool
	}{
		{
			name: "Test_leapArray_currentBucketWithTime_normal",
			fields: fields{
				bucketLengthInMs: BucketLengthInMs,
				sampleCount:      SampleCount,
				intervalInMs:     IntervalInMs,
				array:            newAtomicBucketWrapArray(int(SampleCount), BucketLengthInMs, &leapArrayMock{}),
				mux:              mutex{},
			},
			args: args{
				bg:         new(leapArrayMock),
				timeMillis: 1576296044907,
			},
			want:    nil,
			wantErr: false,
		},
	}

	wwPtr := tests[0].fields.array.get(9)
	wwPtr.bucketStart = 1576296044500 //start time of cycle 1576296040000
	tests[0].want = tests[0].fields.array.get(9)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			la := &leapArray{
				bucketLengthInMs: tt.fields.bucketLengthInMs,
				sampleCount:      tt.fields.sampleCount,
				intervalInMs:     tt.fields.intervalInMs,
				array:            tt.fields.array,
				updateLock:       tt.fields.mux,
			}
			got, err := la.currentBucketOfTime(tt.args.timeMillis, tt.args.bg)
			if (err != nil) != tt.wantErr {
				t.Errorf("leapArray.currentBucketOfTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("leapArray.currentBucketOfTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_leapArray_valuesWithTime_normal(t *testing.T) {
	type fields struct {
		bucketLengthInMs uint32
		sampleCount      uint32
		intervalInMs     uint32
		array            *atomicBucketWrapArray
		mux              mutex
	}
	type args struct {
		timeMillis uint64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *bucketWrap
		wantErr bool
	}{
		{
			name: "Test_leapArray_valuesWithTime_normal",
			fields: fields{
				bucketLengthInMs: BucketLengthInMs,
				sampleCount:      SampleCount,
				intervalInMs:     IntervalInMs,
				array:            newAtomicBucketWrapArray(int(SampleCount), BucketLengthInMs, &leapArrayMock{}),
				mux:              mutex{},
			},
			args: args{
				timeMillis: 1576296049907,
			},
			want:    nil,
			wantErr: false,
		},
	}
	// override start time
	start := uint64(1576296040000)
	for idx := 0; idx < tests[0].fields.array.length; idx++ {
		ww := tests[0].fields.array.get(idx)
		ww.bucketStart = start
		start += 500
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			la := &leapArray{
				bucketLengthInMs: tt.fields.bucketLengthInMs,
				sampleCount:      tt.fields.sampleCount,
				intervalInMs:     tt.fields.intervalInMs,
				array:            tt.fields.array,
				updateLock:       tt.fields.mux,
			}
			got := la.valuesWithTime(tt.args.timeMillis)
			for _, g := range got {
				find := false
				for i := 0; i < tests[0].fields.array.length; i++ {
					w := tests[0].fields.array.get(i)
					if w.bucketStart == g.bucketStart {
						find = true
						break
					}
				}
				if !find {
					t.Errorf("leapArray.valuesWithTime() fail")
				}
			}
		})
	}
}

func Test_leapArray_isBucketDeprecated_normal(t *testing.T) {
	type fields struct {
		bucketLengthInMs uint32
		sampleCount      uint32
		intervalInMs     uint32
		array            *atomicBucketWrapArray
		mux              mutex
	}
	type args struct {
		startTime uint64
		ww        *bucketWrap
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Test_leapArray_isBucketDeprecated_normal",
			fields: fields{
				bucketLengthInMs: BucketLengthInMs,
				sampleCount:      SampleCount,
				intervalInMs:     IntervalInMs,
				array:            newAtomicBucketWrapArray(int(SampleCount), BucketLengthInMs, &leapArrayMock{}),
				mux:              mutex{},
			},
			args: args{
				startTime: 1576296044907,
				ww: &bucketWrap{
					bucketStart: 1576296004907,
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			la := &leapArray{
				bucketLengthInMs: tt.fields.bucketLengthInMs,
				sampleCount:      tt.fields.sampleCount,
				intervalInMs:     tt.fields.intervalInMs,
				array:            tt.fields.array,
				updateLock:       tt.fields.mux,
			}
			if got := la.isBucketDeprecated(tt.args.startTime, tt.args.ww); got != tt.want {
				t.Errorf("leapArray.isBucketDeprecated() = %v, want %v", got, tt.want)
			}
		})
	}
}
