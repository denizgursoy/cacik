Feature: String parameter types

  Scenario: Quoted strings with {string}
    Given the user says "Hello World"
    And the user says "Testing 123!"
    And the user says "Special chars: @#$%"

  Scenario: Empty string
    Given the user says ""

  Scenario: Error messages
    Given the error message is "File not found"
    And the error message is "Connection timeout"
