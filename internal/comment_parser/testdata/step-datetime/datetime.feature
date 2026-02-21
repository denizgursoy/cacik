Feature: Date, Time, DateTime, and Timezone parameter types

  # ===================
  # TIME FORMAT TESTS
  # ===================

  Scenario: Time in 24-hour format
    Given the meeting is at 14:30
    And the meeting is at 09:15
    And the meeting is at 00:00
    And the meeting is at 23:59

  Scenario: Time with seconds
    Given the meeting is at 14:30:45
    And the meeting is at 09:15:00
    And the meeting is at 23:59:59

  Scenario: Time with milliseconds
    Given the meeting is at 14:30:45.123
    And the meeting is at 09:15:00.500
    And the meeting is at 23:59:59.999

  Scenario: Time with AM/PM
    Given the meeting is at 2:30pm
    And the meeting is at 9:15am
    And the meeting is at 12:00pm
    And the meeting is at 12:00am
    And the meeting is at 2:30 PM
    And the meeting is at 9:15 AM

  Scenario: Time with timezone offset
    Given the meeting is at 14:30Z
    And the meeting is at 14:30+05:30
    And the meeting is at 14:30-08:00
    And the meeting is at 14:30+0530
    And the meeting is at 09:15:00+00:00
    And the meeting is at 2:30pm-05:00

  Scenario: Time with IANA timezone
    Given the meeting is at 14:30 Europe/London
    And the meeting is at 09:15 America/New_York
    And the meeting is at 23:00 Asia/Tokyo
    And the meeting is at 2:30pm Europe/Paris
    And the meeting is at 10:00am UTC

  Scenario: Time range
    Given the store is open between 9:00 and 21:00
    And the store is open between 9:00am and 9:00pm

  # ===================
  # DATE FORMAT TESTS (EU DEFAULT)
  # ===================

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

  # ===================
  # DATETIME FORMAT TESTS
  # ===================

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

  # ===================
  # TIMEZONE STANDALONE TESTS
  # ===================

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

  Scenario: Convert datetime to timezone
    Given convert 2024-01-15T14:30:00Z to Europe/London
    And convert 2024-01-15T14:30:00Z to America/New_York
    And convert 2024-06-15 09:00+05:30 to UTC

  # ===================
  # COMBINED FORMAT TESTS
  # ===================

  Scenario: Schedule with date and time
    Given schedule from 2024-01-15 at 9:00 to 2024-01-15 at 17:00
    And schedule from 15/01/2024 at 9:00am to 15/01/2024 at 5:00pm

  Scenario: Tasks with count, date, and time
    Given I have 5 tasks due on 2024-01-15 at 17:00
    And I have 10 tasks due on 31/01/2024 at 11:59pm

  Scenario: Event with name and datetime
    Given event "Team Meeting" starts at 2024-01-15 10:00
    And event "Product Launch" starts at 2024-06-01T09:00 Europe/London

  Scenario: Meeting with time and explicit timezone
    Given meeting at 14:30 in Europe/London
    And meeting at 9:00am in America/New_York
    And meeting at 18:00 in Asia/Tokyo

  # ===================
  # EDGE CASES
  # ===================

  Scenario: Midnight and noon
    Given the meeting is at 00:00
    And the meeting is at 12:00
    And the meeting is at 12:00am
    And the meeting is at 12:00pm

  Scenario: End of day
    Given the meeting is at 23:59:59
    And the meeting is at 23:59:59.999

  Scenario: Leap year date
    Given the event is on 29/02/2024
    And the event is on 2024-02-29

  Scenario: Single digit day and month
    Given the event is on 1/2/2024
    And the event is on 01/02/2024
    And the event is on 1 Feb 2024
