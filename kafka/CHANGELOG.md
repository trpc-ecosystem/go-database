# Change Log

## [kafka/v1.2.0](https://github.com/trpc-ecosystem/go-database/releases/tag/kafka%2Fv1.2.0) (2024-12-11)

### Features

- kafka: change default metadataRefreshFrequency to 60s and use GroupStrategies instead of deprecated Strategy config ( sync from the internal merge request 1379)  (#34)
- kafka: allow users to set sarama's Metadata parameters(sync from the internal merge request 1442) (#33)

## [kafka/v1.1.0](https://github.com/trpc-ecosystem/go-database/releases/tag/kafka%2Fv1.1.0) (2023-12-22)

### Breaking Changes

- update sarama dependence from to github.com/Shopify/sarama v1.29.1 to github.com/IBM/sarama v1.40.1 (#21)

### Bug Fixes

- fix unit test (#21)