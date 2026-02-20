Feature: Color selection with custom type

  Scenario: Select red color
    When I select red
    Then the color is red

  Scenario: Select blue color
    When I select blue
    Then the color is blue

  Scenario: Select green color
    When I select green
    Then the color is green

  Scenario: Case insensitive matching
    When I select RED
    Then the color is red
