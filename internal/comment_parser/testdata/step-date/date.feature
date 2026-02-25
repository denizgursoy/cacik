Feature: Date parameter type

  Scenario: Date in EU format (DD/MM/YYYY)
    Given the event is on 15/01/2024
    And the event is on 31/12/2024
    And the event is on 01/06/2025

  Scenario: Date in EU format with different separators
    Given the event is on 15-01-2024
    And the event is on 15.01.2024
    And the event is on 31.12.2024

  Scenario: Date in ISO format (YYYY-MM-DD)
    Given the event is on 2024-01-15
    And the event is on 2024-12-31
    And the event is on 2025-06-01

  Scenario: Date with slashes ISO (YYYY/MM/DD)
    Given the event is on 2024/01/15
    And the event is on 2024/12/31

  Scenario: Date in written format
    Given the event is on 15 Jan 2024
    And the event is on 15 January 2024
    And the event is on 31 Dec 2024
    And the event is on 1 June 2025

  Scenario: Date in written format (Month first)
    Given the event is on Jan 15, 2024
    And the event is on January 15, 2024
    And the event is on Dec 31, 2024

  Scenario: Date range
    Given the sale runs from 2024-01-01 to 2024-12-31
    And the sale runs from 01/01/2024 to 31/12/2024
    And the sale runs from 1 Jan 2024 to 31 Dec 2024

  Scenario: Leap year date
    Given the event is on 29/02/2024
    And the event is on 2024-02-29

  Scenario: Single digit day and month
    Given the event is on 1/2/2024
    And the event is on 01/02/2024
    And the event is on 1 Feb 2024
