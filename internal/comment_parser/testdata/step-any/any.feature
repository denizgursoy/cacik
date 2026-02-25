Feature: Any parameter type

  Scenario: Match anything
    Given I see anything at all here
    And I see 123 mixed content!
    And I see special chars: @#$% and more

  Scenario: Free-form descriptions
    Given the description is a long text with spaces and punctuation!
    And the description is 42
