Feature: Time parameter type

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

  Scenario: Midnight and noon
    Given the meeting is at 00:00
    And the meeting is at 12:00
    And the meeting is at 12:00am
    And the meeting is at 12:00pm

  Scenario: End of day
    Given the meeting is at 23:59:59
    And the meeting is at 23:59:59.999
