Feature: Timezone parameter type

  Scenario: Timezone as UTC/Z
    Given convert to UTC
    And convert to Z

  Scenario: Timezone as offset
    Given convert to +00:00
    And convert to +05:30
    And convert to -08:00
    And convert to +0530
    And convert to -0800

  Scenario: Timezone as IANA name
    Given convert to Europe/London
    And convert to America/New_York
    And convert to America/Los_Angeles
    And convert to Asia/Tokyo
    And convert to Asia/Kolkata
    And convert to Australia/Sydney

  Scenario: Show time in different timezones
    Given show current time in UTC
    And show current time in Europe/London
    And show current time in America/New_York
    And show current time in Asia/Tokyo
