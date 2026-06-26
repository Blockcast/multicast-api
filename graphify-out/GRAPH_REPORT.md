# Graph Report - multicast-api  (2026-06-26)

## Corpus Check
- 39 files · ~24,590 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 972 nodes · 1557 edges · 89 communities (68 shown, 21 thin omitted)
- Extraction: 98% EXTRACTED · 2% INFERRED · 0% AMBIGUOUS · INFERRED: 36 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `9b3ad8b3`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- [[_COMMUNITY_Community 0|Community 0]]
- [[_COMMUNITY_Community 1|Community 1]]
- [[_COMMUNITY_Community 2|Community 2]]
- [[_COMMUNITY_Community 3|Community 3]]
- [[_COMMUNITY_Community 4|Community 4]]
- [[_COMMUNITY_Community 5|Community 5]]
- [[_COMMUNITY_Community 6|Community 6]]
- [[_COMMUNITY_Community 7|Community 7]]
- [[_COMMUNITY_Community 8|Community 8]]
- [[_COMMUNITY_Community 9|Community 9]]
- [[_COMMUNITY_Community 10|Community 10]]
- [[_COMMUNITY_Community 11|Community 11]]
- [[_COMMUNITY_Community 12|Community 12]]
- [[_COMMUNITY_Community 13|Community 13]]
- [[_COMMUNITY_Community 14|Community 14]]
- [[_COMMUNITY_Community 15|Community 15]]
- [[_COMMUNITY_Community 16|Community 16]]
- [[_COMMUNITY_Community 17|Community 17]]
- [[_COMMUNITY_Community 18|Community 18]]
- [[_COMMUNITY_Community 19|Community 19]]
- [[_COMMUNITY_Community 20|Community 20]]
- [[_COMMUNITY_Community 21|Community 21]]
- [[_COMMUNITY_Community 22|Community 22]]
- [[_COMMUNITY_Community 23|Community 23]]
- [[_COMMUNITY_Community 24|Community 24]]
- [[_COMMUNITY_Community 25|Community 25]]
- [[_COMMUNITY_Community 26|Community 26]]
- [[_COMMUNITY_Community 27|Community 27]]
- [[_COMMUNITY_Community 28|Community 28]]
- [[_COMMUNITY_Community 29|Community 29]]
- [[_COMMUNITY_Community 30|Community 30]]
- [[_COMMUNITY_Community 31|Community 31]]
- [[_COMMUNITY_Community 32|Community 32]]
- [[_COMMUNITY_Community 33|Community 33]]
- [[_COMMUNITY_Community 34|Community 34]]
- [[_COMMUNITY_Community 35|Community 35]]
- [[_COMMUNITY_Community 36|Community 36]]
- [[_COMMUNITY_Community 37|Community 37]]
- [[_COMMUNITY_Community 38|Community 38]]
- [[_COMMUNITY_Community 39|Community 39]]
- [[_COMMUNITY_Community 40|Community 40]]
- [[_COMMUNITY_Community 41|Community 41]]
- [[_COMMUNITY_Community 42|Community 42]]
- [[_COMMUNITY_Community 43|Community 43]]
- [[_COMMUNITY_Community 44|Community 44]]
- [[_COMMUNITY_Community 45|Community 45]]
- [[_COMMUNITY_Community 46|Community 46]]
- [[_COMMUNITY_Community 47|Community 47]]
- [[_COMMUNITY_Community 48|Community 48]]
- [[_COMMUNITY_Community 49|Community 49]]
- [[_COMMUNITY_Community 50|Community 50]]
- [[_COMMUNITY_Community 51|Community 51]]
- [[_COMMUNITY_Community 52|Community 52]]
- [[_COMMUNITY_Community 53|Community 53]]
- [[_COMMUNITY_Community 54|Community 54]]
- [[_COMMUNITY_Community 55|Community 55]]
- [[_COMMUNITY_Community 56|Community 56]]
- [[_COMMUNITY_Community 57|Community 57]]
- [[_COMMUNITY_Community 58|Community 58]]
- [[_COMMUNITY_Community 59|Community 59]]
- [[_COMMUNITY_Community 60|Community 60]]
- [[_COMMUNITY_Community 61|Community 61]]
- [[_COMMUNITY_Community 62|Community 62]]
- [[_COMMUNITY_Community 63|Community 63]]
- [[_COMMUNITY_Community 64|Community 64]]
- [[_COMMUNITY_Community 65|Community 65]]
- [[_COMMUNITY_Community 66|Community 66]]
- [[_COMMUNITY_Community 67|Community 67]]
- [[_COMMUNITY_Community 68|Community 68]]
- [[_COMMUNITY_Community 69|Community 69]]
- [[_COMMUNITY_Community 70|Community 70]]
- [[_COMMUNITY_Community 71|Community 71]]
- [[_COMMUNITY_Community 72|Community 72]]
- [[_COMMUNITY_Community 73|Community 73]]
- [[_COMMUNITY_Community 74|Community 74]]
- [[_COMMUNITY_Community 84|Community 84]]

## God Nodes (most connected - your core abstractions)
1. `StartElement` - 49 edges
2. `Decoder` - 48 edges
3. `DurationType` - 21 edges
4. `RangeList` - 20 edges
5. `UserServiceDescription` - 18 edges
6. `TextualType` - 18 edges
7. `ControlledTermUseType` - 15 edges
8. `BlockingStructure` - 15 edges
9. `ClassificationPreferencesType` - 14 edges
10. `Lang` - 13 edges

## Surprising Connections (you probably didn't know these)
- `ESRangeListForSBNFromRangeList()` --calls--> `Uint32`  [INFERRED]
  fec/util.go → fec/blocking.go
- `ESIRangeFromRangeList()` --calls--> `Uint64`  [INFERRED]
  fec/util.go → fec/blocking.go
- `TestESI()` --calls--> `NewESIRange()`  [INFERRED]
  fec/util_test.go → fec/util.go
- `TestESIRangeFromRangeList()` --calls--> `GetMissingESIs()`  [INFERRED]
  fec/util_test.go → fec/util.go
- `TestESIRangeFromRangeListBorderCase()` --calls--> `GetMissingESIs()`  [INFERRED]
  fec/util_test.go → fec/util.go

## Import Cycles
- None detected.

## Communities (89 total, 21 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.04
Nodes (57): DASHComponentIdentifierType, HLSComponentIdentifierType, MimeType, PresentationManifestLocator, Type, MediaTimeType, ActionTime, Anon1 (+49 more)

### Community 1 - "Community 1"
Cohesion: 0.09
Nodes (24): Attr, Decoder, Encoder, xsdBase64Binary, Name, StartElement, Time, FileURI (+16 more)

### Community 3 - "Community 3"
Cohesion: 0.08
Nodes (38): Creator, Domain, DurationType, BitRateType, ContentAcquisitionMethodType, TransmissionModeType, TransportSecurityType, ForwardErrorCorrectionParametersType (+30 more)

### Community 4 - "Community 4"
Cohesion: 0.09
Nodes (30): Type, Identifier, LinearRing, Coord, CoordinateReferenceSystem, EllipticalArea, Emetrigger, Eqop (+22 more)

### Community 5 - "Community 5"
Cohesion: 0.09
Nodes (11): Frequency, Freq, RRule, Weekday, Set, Attr, Duration, Duration (+3 more)

### Community 6 - "Community 6"
Cohesion: 0.07
Nodes (26): Coverage Validation via SONAR, FCI Capability Advertisement, Footprint to DeliveryMethod, German Tank Problem, Import OC-API Types, MI.OcnSelection Properties, Named Footprints (draft-ietf-cdni-named-footprints), OC-API Type Mapping (+18 more)

### Community 7 - "Community 7"
Cohesion: 0.07
Nodes (10): CarouselMode, ContentAcquisitionMethodType, DeliveryMode, FileStatus, SessionType, StoreType, TransmissionModeType, TransportProtocolType (+2 more)

### Community 8 - "Community 8"
Cohesion: 0.13
Nodes (8): NewBlockingStructure5052(), UpdateBlockingStructure5052(), BlockingStructure, PserviceArea, Uint32, Uint64, Attr, AtomicUint32

### Community 9 - "Community 9"
Cohesion: 0.12
Nodes (15): Decoder, Time, StartElement, Type, Esrd, Esrk, Lastclient, Loctype (+7 more)

### Community 10 - "Community 10"
Cohesion: 0.09
Nodes (21): CarouselMode, FilePull, FileStatus, FilesType, AMTRelayConfig, FilePull, FilesType, Service (+13 more)

### Community 11 - "Community 11"
Cohesion: 0.09
Nodes (22): Addr, CodePoint, ContentAcquisitionMethodType, DASHComponents, ChannelDesc(), BitRateType, Duration, BitRateType (+14 more)

### Community 12 - "Community 12"
Cohesion: 0.12
Nodes (9): BitRateType, DASHComponents, FECEncoding, FECParamsType, FECParamType, HLSComponents, MulticastEndpointAddressesType, MulticastEndpointAddressType (+1 more)

### Community 13 - "Community 13"
Cohesion: 0.10
Nodes (19): Decoder, StartElement, BasicProcedureType, BmFileRepairType, ConsumptionReportType, DASHQoEProcedureType, AssociatedProcedureType, BasicProcedureType (+11 more)

### Community 14 - "Community 14"
Cohesion: 0.14
Nodes (11): caseList, feedOfSub, caseList, FeedOf, FeedOf[T], feedOfSub, feedOfSub[T], Subscription (+3 more)

### Community 15 - "Community 15"
Cohesion: 0.17
Nodes (21): MediaLocatorType, Affiliation, AgentType, ControlledTermUseType, CountryCode, Creator, CreatorType, Disseminator (+13 more)

### Community 16 - "Community 16"
Cohesion: 0.18
Nodes (11): Decoder, Encoder, StartElement, MediaFlow, FecProtectionType, KeyIdType, KeyManagementType, MediaFlow (+3 more)

### Community 17 - "Community 17"
Cohesion: 0.12
Nodes (17): Name, DASHContent, Frequency, IdenticalContent, InfoBinding, InitiationRandomization, KeepUpdatedService, MediaPresentationDescription (+9 more)

### Community 18 - "Community 18"
Cohesion: 0.16
Nodes (18): Name, Name, Anon15, ColorDomain, DisseminationFormat, Form, Format, Genre (+10 more)

### Community 19 - "Community 19"
Cohesion: 0.15
Nodes (17): Type, Lang, Anon11, Anon13, ExtendedLanguageType, Keyword, Lang, Language (+9 more)

### Community 20 - "Community 20"
Cohesion: 0.22
Nodes (6): F, RangeList, ESRangeListForSBNFromRangeList(), flatten(), MakeRandRangeList(), FuzzRangeList_SetOps()

### Community 21 - "Community 21"
Cohesion: 0.12
Nodes (17): InlineMediaType, MediaTimePointType, MimeType, Anon4, BytePosition, ImageLocatorType, InlineMediaType, MediaDurationType (+9 more)

### Community 22 - "Community 22"
Cohesion: 0.14
Nodes (15): Polygon, CircularArcArea, GroupFilter, LocationFilter, LocationRule, LogicalOperation, FilterData, FilterDescription (+7 more)

### Community 23 - "Community 23"
Cohesion: 0.23
Nodes (15): Missing(), ParseContentRange(), RangeList, T, parseRangeList(), TestContentRange(), TestESI(), TestESIRangeFromRangeList() (+7 more)

### Community 24 - "Community 24"
Cohesion: 0.22
Nodes (15): Time, GeographicPosition, AdministrativeUnit, DisseminationLocation, GeographicPointType, GeographicPosition, Location, ParentalGuidance (+7 more)

### Community 25 - "Community 25"
Cohesion: 0.14
Nodes (14): DeliveryMethod, Name, InfoBinding, KeepUpdatedService, MediaPresentationDescription, AccessGroup, AvailabilityInfo, ConsumptionReporting (+6 more)

### Community 26 - "Community 26"
Cohesion: 0.23
Nodes (14): Time, Anon5, DurationType, IncrDurationType, Recurrence, RelIncrTimePointType, RelTimePointType, Term (+6 more)

### Community 27 - "Community 27"
Cohesion: 0.16
Nodes (14): Cgi, Gsmnetparam, Pos, Result, Slia, Slirep, Slrep, Svcresult (+6 more)

### Community 28 - "Community 28"
Cohesion: 0.15
Nodes (11): Country, DatePeriod, Form, Genre, Language, LanguageFormat, CaptionLanguage, ClassificationPreferencesType (+3 more)

### Community 29 - "Community 29"
Cohesion: 0.18
Nodes (12): Time, Emepos, Emetrigger, Emeevent, Emelia, Emerep, Pd, Poserr (+4 more)

### Community 31 - "Community 31"
Cohesion: 0.17
Nodes (9): FilteringAndSearchPreferencesType, UsageHistoryType, UserActionHistoryType, PreferenceConditionType, SourcePreferencesType, UserActionHistoryType, UserActionListType, UserChoiceType (+1 more)

### Community 32 - "Community 32"
Cohesion: 0.29
Nodes (6): BlockingStructure, BlockReadError, ESIRangeFromRangeList(), GetMissingESIs(), RangeList, NewBlockReadError()

### Community 33 - "Community 33"
Cohesion: 0.22
Nodes (11): ControlledTermUseType, Emphasis, AudioChannels, AudioCoding, BitRate, DisseminationSource, MediaFormat, MediaFormatType (+3 more)

### Community 34 - "Community 34"
Cohesion: 0.24
Nodes (5): MimeType, PresentationManifestLocator, Value, quoteIfNeeded(), unquoteIfNeeded()

### Community 35 - "Community 35"
Cohesion: 0.20
Nodes (3): ESIRange, NewESIRange(), NewESIRangeFromMBMSQuery()

### Community 36 - "Community 36"
Cohesion: 0.24
Nodes (4): FilePulls, JSONStruct, StringSlice, Value

### Community 37 - "Community 37"
Cohesion: 0.25
Nodes (7): Name, BlockcastFileURI, BlockcastStatisticalReport, BlockcastFileURI, BlockcastReceptionReport, BlockcastStatisticalReport, FState

### Community 38 - "Community 38"
Cohesion: 0.22
Nodes (9): Enc, Msid, Msidrange, Msids, Startmsid, Stopmsid, Session, Startmsid (+1 more)

### Community 40 - "Community 40"
Cohesion: 0.38
Nodes (4): Decoder, StartElement, AlternativeAccessDelivery, Registration

### Community 41 - "Community 41"
Cohesion: 0.29
Nodes (5): DASHComponentIdentifierType, Decoder, StartElement, HLSComponentIdentifierType, ServiceComponentIdentifierType

### Community 42 - "Community 42"
Cohesion: 0.29
Nodes (3): DASHComponentIdentifierType, HLSComponentIdentifierType, Value

### Community 43 - "Community 43"
Cohesion: 0.29
Nodes (4): Duration, RRuleSet, TimeZ, Value

### Community 44 - "Community 44"
Cohesion: 0.47
Nodes (6): Polygon, Coord, Box, CircularArcArea, CircularArea, MultiPolygon

### Community 45 - "Community 45"
Cohesion: 0.33
Nodes (4): BlockRangeReadError, ESIRange, BlockRangeReadError, NewBlockRangeReadError()

### Community 46 - "Community 46"
Cohesion: 0.33
Nodes (6): Esrd, Esrk, Emepos, Msid, Pd, Poserr

### Community 47 - "Community 47"
Cohesion: 0.33
Nodes (3): FECEncoding, CodePoint, FECInstance

### Community 48 - "Community 48"
Cohesion: 0.33
Nodes (6): BroadcastAppService, DeliveryMethod, PserviceArea, ROMSvcRfParams, SupplementaryUnicastAppService, UnicastAppService

### Community 49 - "Community 49"
Cohesion: 0.40
Nodes (4): PresentationManifestLocator, MulticastSessionType, MulticastTransportSessionType, SessionReportingType

### Community 50 - "Community 50"
Cohesion: 0.40
Nodes (5): Client, Hdr, Requestmode, Requestor, Subclient

### Community 51 - "Community 51"
Cohesion: 0.50
Nodes (5): TextualType, Tool, UserChoiceType, UserIdentifierType, UserPreferencesType

### Community 53 - "Community 53"
Cohesion: 0.67
Nodes (3): T, TestFECParamsScanCanonicalDBText(), TestFECParamsScanDualStackEndpointValues()

### Community 54 - "Community 54"
Cohesion: 0.67
Nodes (3): T, TestFECEncodingScan(), TestFECEncodingScanErrors()

### Community 56 - "Community 56"
Cohesion: 0.50
Nodes (4): InitiationRandomization, BundleDescription, TerminationRandomization, UserServiceDescription

### Community 57 - "Community 57"
Cohesion: 0.50
Nodes (3): BrowsingPreferencesType, PreferenceCondition, SummaryPreferencesType

### Community 62 - "Community 62"
Cohesion: 0.67
Nodes (3): IdenticalContent, AlternativeContent, AppService

### Community 65 - "Community 65"
Cohesion: 0.67
Nodes (3): Neid, Vlrid, Vmscid

## Knowledge Gaps
- **268 isolated node(s):** `BasicProcedureType`, `BmFileRepairType`, `ReportProcedureType`, `ConsumptionReportType`, `DASHQoEProcedureType` (+263 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **21 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Uint64` connect `Community 8` to `Community 32`, `Community 35`, `Community 12`?**
  _High betweenness centrality (0.022) - this node is a cross-community bridge._
- **Why does `PserviceArea` connect `Community 8` to `Community 17`?**
  _High betweenness centrality (0.018) - this node is a cross-community bridge._
- **Why does `ESIRangeFromRangeList()` connect `Community 32` to `Community 8`, `Community 35`, `Community 20`, `Community 23`?**
  _High betweenness centrality (0.016) - this node is a cross-community bridge._
- **What connects `BasicProcedureType`, `BmFileRepairType`, `ReportProcedureType` to the rest of the system?**
  _268 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.0411373260738052 - nodes in this community are weakly interconnected._
- **Should `Community 1` be split into smaller, more focused modules?**
  _Cohesion score 0.09024390243902439 - nodes in this community are weakly interconnected._
- **Should `Community 2` be split into smaller, more focused modules?**
  _Cohesion score 0.0951219512195122 - nodes in this community are weakly interconnected._