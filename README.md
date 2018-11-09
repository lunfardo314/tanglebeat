## About Tanglebeat
**Tanglebeat** is a lightweight yet highly configurable software agent with the main purpose to collect Tangle-related metrics to 
[Prometheus TSDB](https://prometheus.io/) to be displayed with such tools
as [Grafana](https://grafana.com). 

It can be run in various standalone and distributed configurations to ensure 
high availability and objectivty of the metrics.

Demo Grafana dashboard with testing configuration behind (two instances of Tangelbeat) can be found at [tanglebeat.com](http://tanglebeat.com:3000/d/85B_28aiz/tanglebeat-demo?refresh=10s&orgId=1&from=1541747401598&to=1541769001598&tab=general)

Tanglebeat is a successor of [Traveling IOTA](http://traviota.iotalt.com) project, 
scaled up and, hopefully, more practical version of the latter.

## Transfer confirmation metrics

Tanglebeat is performing IOTA value transfers from one address 
to the next one in the endless loop.  
Tangle beat makes sure the bundle of the transfer is confirmed by _promoting_
it and _reattaching_ (if necessary). 
Tanglebeat starts sending 
whole balance of iotas to the next address in the sequence immediately after current transfer is confirmed. And so on.

Several sequences of addresses are run in parallel. 
Confirmation time and other statistical data is collected in 
the process of sending and, after some averaging, is provided as 
metrics. 

Examples of the metrics are _transfers per hour or TfPH_, _average PoW cost per confirmed transfer_, _average confirmation time_.

Tanglebeat publishes much more data about transfers thorough the open message channel. It can be used to calculate other metrics and to visualize transfer progress.

## ZeroMQ metrics

Tanglebeat also provides usual metrics derived from data of Zero MQ stream by IRI such as _TPS_, _CTPS_, _Confirmation rate_ and _duration between milestones_

## Highly configurable

Each Tanglebeat agent consist of the following functional 
parts. Every part can be enabled, disabled and configured
independently from each other thus enabling congfiguration of any size and complexity.
- Sender
- Update collector
- Prometheus metrics collectors. It consists _sender metrics_ part and _ZMQ metrics_ part.

Tanglebeat is single binary. It takes no command line argumenst. Each Tanglebeat instance is configured 
through  [tanglebeat.yml](tanglebeat.yml). File which must be located in the working 
directory of the instance. It contains seeds of sequences therefore should never be made public.

#### Sender

_Sender_ is running sequences of transfers. It generates transfer bundles, promotion and 
(re)attaches them until confirmed for each of enabled sequences of addresses. 
Sender is configured through `sender` section in the config file. It contains global and individual parameters 
for each sequence of transfers.
```
sender:
    enabled: true
    globals:
        iotaNode: &DEFNODE https://field.deviota.com:443    
        iotaNodeTipsel: *DEFNODE              
        iotaNodePOW: https://api.powsrv.io:443
        apiTimeout: 15
        tipselTimeout: 15
        powTimeout: 15
        txTag: TANGLE9BEAT
        txTagPromote: TANGLE9BEAT
        forceReattachAfterMin: 15
        promoteEverySec: 10
        
    sequences:
        Sequence 1:
            enabled: true
            iotaNode: http://node.iotalt.com:14600
            seed: SEED99999999999999
            index0: 0
            promoteChain: true
        Another sequence:
            enabled: false
            seed: SEED9ANOTHER9999999999999
            index0: 0
            promoteChain: true
        # more sequences can be configured    
```   
Sender generates **updates** with the information 
about the progress, per sequence. Each update [contains all the data](https://github.com/lunfardo314/tanglebeat/blob/baf8c69bc119e5ba854d0d28a8746df94f1d318b/sender_update/types.go#L22), 
about the event. Metrics are calculated from sender updates. 
It also allows visualisation of the sendinf process like [Traveling IOTA](http://traviota.iotalt.com) does.
 
#### Update collector

If enabled, update collector gathers updates from one or many 
senders (sources) into one resulting stream of updates. This function allows to configure distributed network of Tanglebeat agents.
The sender itself is on of sources.
Data stream from update collector is used to calculate metrics for Prometheus.

```
senderUpdateCollector: 
    # sender update collector is enabled if at least one of sources is enabled
    # sender update collector collects updates from one or more sources to one stream
    sources: 
        # sources with the name 'local' means that all updates generated by this instance are collected into 
        # resulting stream. If 'local' source is disabled, it means that localy generated updates have no effect on metrics and are not published
        local:
            enabled: true
        # unlimited number of external tanglebeat instances can be specified. 
        # Each source is listened for published updates and collected into the resulting stream. 
        tanglebeat2:
            enabled: true
            target: "tcp://node.iotalt.com:3100"
    # if true, resulting stream of updates is published to outside through specified port
    # if false, resulting stream is not published and only used to calculate the metrics
    publish: true
    outPort: 3100
```

If `publish=true` resulting stream of updates is exposed through specified ports and can be collected by other 
Tanglebeat instances. Data is published over [Nanomsg/Mangos](https://github.com/nanomsg/mangos) 
sockets in the form of [JSON messages](https://github.com/lunfardo314/tanglebeat/blob/baf8c69bc119e5ba854d0d28a8746df94f1d318b/sender_update/types.go#L22).

Published sender updates can be subscribed by external consumers for expample in order to calculate own metrics. 

[Here's an example](https://github.com/lunfardo314/tanglebeat/tree/ver0/examples/statsws) of a client (in Go) which calculates averaged confirmation time statistics to expose it as web service


#### Prometheus metrics collectors
If enabled, it exposes metrics to Prometheus. There are two 
independent parts:
- _Sender metrics_. It exposes metrics calculated from sender update stream: 
    - `tanglebeat_confirmation_counter`, `tanglebeat_pow_cost_counter`, `anglebeat_confirmation_duration_counter`,
    `tanglebeat_pow_duration_counter`, `tanglebeat_tipsel_duration_counter`
    - and metrics precalculated by Prometheus rules in [tanglebeat.rules](tanglebeat.rules)

- _Zero MQ metrics_. If enabled, reads Zero MQ from IRI node, calculates and exposes 
   TPS, CTPS etc metrics to Prometheus.

