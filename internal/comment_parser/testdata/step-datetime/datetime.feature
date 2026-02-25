Feature: DateTime parameter type

  Scenario: DateTime in ISO format with space
    Given the appointment is at 2024-01-15 14:30
    And the appointment is at 2024-12-31 23:59:59
    And the appointment is at 2024-06-15 09:00:00.500

  Scenario: DateTime in ISO format with T separator
    Given the appointment is at 2024-01-15T14:30
    And the appointment is at 2024-12-31T23:59:59
    And the appointment is at 2024-06-15T09:00:00.123

  Scenario: DateTime with AM/PM
    Given the appointment is at 2024-01-15 2:30pm
    And the appointment is at 2024-12-31 11:59pm
    And the appointment is at 15/01/2024 9:00am

  Scenario: DateTime with timezone Z (UTC)
    Given the flight departs at 2024-01-15T14:30:00Z
    And the flight departs at 2024-12-31T23:59:59Z
    And the flight departs at 2024-06-15 09:00Z

  Scenario: DateTime with timezone offset
    Given the flight departs at 2024-01-15T14:30:00+05:30
    And the flight departs at 2024-12-31T23:59:59-08:00
    And the flight departs at 2024-06-15 09:00+00:00
    And the flight departs at 15/01/2024 14:30+0530

  Scenario: DateTime with IANA timezone
    Given the flight departs at 2024-01-15 14:30 Europe/London
    And the flight departs at 2024-12-31 23:59 America/New_York
    And the flight departs at 2024-06-15 09:00 Asia/Tokyo
    And the flight departs at 15/01/2024 2:30pm Europe/Paris
