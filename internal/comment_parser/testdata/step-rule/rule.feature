Feature: User management

  Background:
    Given the system is initialized

  Rule: User registration

    Background:
      Given the registration form is loaded

    Scenario: Register with valid email
      When the user registers with "alice@example.com"
      Then the registration should succeed

    Scenario: Register with invalid email
      When the user registers with "not-an-email"
      Then the registration should fail

  Rule: User login

    Background:
      Given the login page is loaded

    Scenario: Login with valid credentials
      When the user logs in with "alice" and "secret"
      Then the login should succeed

    Scenario: Login with wrong password
      When the user logs in with "alice" and "wrong"
      Then the login should fail
