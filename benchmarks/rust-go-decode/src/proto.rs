// Copyright (c) Bartłomiej Płotka @bwplotka
// Licensed under the Apache License 2.0.

//! Prost message definitions matching Prometheus Remote Write 2.0 (`io.prometheus.write.v2.Request`).

pub mod v2 {
    #[derive(Clone, PartialEq, ::prost::Message)]
    pub struct Request {
        #[prost(string, repeated, tag = "4")]
        pub symbols: ::prost::alloc::vec::Vec<::prost::alloc::string::String>,
        #[prost(message, repeated, tag = "5")]
        pub timeseries: ::prost::alloc::vec::Vec<TimeSeries>,
    }

    #[derive(Clone, PartialEq, ::prost::Message)]
    pub struct TimeSeries {
        #[prost(uint32, repeated, tag = "1")]
        pub labels_refs: ::prost::alloc::vec::Vec<u32>,
        #[prost(message, repeated, tag = "2")]
        pub samples: ::prost::alloc::vec::Vec<Sample>,
        #[prost(message, repeated, tag = "3")]
        pub histograms: ::prost::alloc::vec::Vec<Histogram>,
        #[prost(message, repeated, tag = "4")]
        pub exemplars: ::prost::alloc::vec::Vec<Exemplar>,
        #[prost(message, optional, tag = "5")]
        pub metadata: ::core::option::Option<Metadata>,
    }

    #[derive(Clone, Copy, PartialEq, ::prost::Message)]
    pub struct Sample {
        #[prost(double, tag = "1")]
        pub value: f64,
        #[prost(int64, tag = "2")]
        pub timestamp: i64,
        #[prost(int64, tag = "3")]
        pub start_timestamp: i64,
    }

    #[derive(Clone, PartialEq, ::prost::Message)]
    pub struct Exemplar {
        #[prost(uint32, repeated, tag = "1")]
        pub labels_refs: ::prost::alloc::vec::Vec<u32>,
        #[prost(double, tag = "2")]
        pub value: f64,
        #[prost(int64, tag = "3")]
        pub timestamp: i64,
    }

    #[derive(Clone, Copy, PartialEq, Eq, ::prost::Message)]
    pub struct Metadata {
        #[prost(int32, tag = "1")]
        pub r#type: i32,
        #[prost(uint32, tag = "3")]
        pub help_ref: u32,
        #[prost(uint32, tag = "4")]
        pub unit_ref: u32,
    }

    #[derive(Clone, PartialEq, ::prost::Message)]
    pub struct Histogram {
        #[prost(oneof = "histogram::Count", tags = "1, 2")]
        pub count: ::core::option::Option<histogram::Count>,
        #[prost(double, tag = "3")]
        pub sum: f64,
        #[prost(sint32, tag = "4")]
        pub schema: i32,
        #[prost(double, tag = "5")]
        pub zero_threshold: f64,
        #[prost(oneof = "histogram::ZeroCount", tags = "6, 7")]
        pub zero_count: ::core::option::Option<histogram::ZeroCount>,
        #[prost(message, repeated, tag = "8")]
        pub negative_spans: ::prost::alloc::vec::Vec<BucketSpan>,
        #[prost(sint64, repeated, tag = "9")]
        pub negative_deltas: ::prost::alloc::vec::Vec<i64>,
        #[prost(double, repeated, tag = "10")]
        pub negative_counts: ::prost::alloc::vec::Vec<f64>,
        #[prost(message, repeated, tag = "11")]
        pub positive_spans: ::prost::alloc::vec::Vec<BucketSpan>,
        #[prost(sint64, repeated, tag = "12")]
        pub positive_deltas: ::prost::alloc::vec::Vec<i64>,
        #[prost(double, repeated, tag = "13")]
        pub positive_counts: ::prost::alloc::vec::Vec<f64>,
        #[prost(int32, tag = "14")]
        pub reset_hint: i32,
        #[prost(int64, tag = "15")]
        pub timestamp: i64,
        #[prost(double, repeated, tag = "16")]
        pub custom_values: ::prost::alloc::vec::Vec<f64>,
        #[prost(int64, tag = "17")]
        pub start_timestamp: i64,
    }

    pub mod histogram {
        #[derive(Clone, Copy, PartialEq, ::prost::Oneof)]
        pub enum Count {
            #[prost(uint64, tag = "1")]
            CountInt(u64),
            #[prost(double, tag = "2")]
            CountFloat(f64),
        }

        #[derive(Clone, Copy, PartialEq, ::prost::Oneof)]
        pub enum ZeroCount {
            #[prost(uint64, tag = "6")]
            ZeroCountInt(u64),
            #[prost(double, tag = "7")]
            ZeroCountFloat(f64),
        }
    }

    #[derive(Clone, Copy, PartialEq, Eq, ::prost::Message)]
    pub struct BucketSpan {
        #[prost(sint32, tag = "1")]
        pub offset: i32,
        #[prost(uint32, tag = "2")]
        pub length: u32,
    }
}
