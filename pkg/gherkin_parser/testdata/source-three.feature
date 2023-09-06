Feature: Scenario Outline

  @Test
  Scenario Outline: Scenario Outline
    Given I run the step with "<parameter1>" and "<parameter2>"
    Examples:
      | parameter1  | parameter2  |
      | parameter1a | parameter2a |
      | parameter1b | parameter2b |
