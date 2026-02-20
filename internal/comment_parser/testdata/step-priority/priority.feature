Feature: Priority levels with custom int type

  Scenario: Set priority by name
    Given priority is low
    And priority is medium
    And priority is high

  Scenario: Set priority by value
    Given priority is 1
    And priority is 2
    And priority is 3

  Scenario: Check priority
    Given priority is high
    Then the priority is high
