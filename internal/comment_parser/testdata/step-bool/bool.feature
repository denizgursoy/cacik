Feature: Boolean parameter types

  Scenario: Using true/false
    Given it is true
    And it is false

  Scenario: Using yes/no
    Given it is yes
    And it is no

  Scenario: Using on/off
    Given it is on
    And it is off

  Scenario: Using enabled/disabled
    Given it is enabled
    And it is disabled

  Scenario: Feature toggle enabled
    Given the feature is enabled

  Scenario: Feature toggle disabled
    Given the feature is disabled

  Scenario: Case insensitive
    Given it is TRUE
    And it is FALSE
    And it is Yes
    And it is NO
