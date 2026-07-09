// @ts-check
const eslint = require('@eslint/js');
const { defineConfig } = require('eslint/config');
const tseslint = require('typescript-eslint');
const angular = require('angular-eslint');
const security = require('eslint-plugin-security');

module.exports = defineConfig([
  {
    files: ['**/*.ts'],
    extends: [
      eslint.configs.recommended,
      tseslint.configs.recommended,
      tseslint.configs.stylistic,
      angular.configs.tsRecommended,
    ],
    processor: angular.processInlineTemplates,
    plugins: {
      security: security,
    },
    rules: {
      ...security.configs.recommended.rules,
      'security/detect-object-injection': 'off', // Turn off noisy object injection checks
      '@typescript-eslint/no-explicit-any': 'off', // Allow any type in legacy codebase
      '@typescript-eslint/no-inferrable-types': 'off', // Allow trivial types
      '@typescript-eslint/array-type': 'off', // Allow ReadonlyArray syntax
      '@typescript-eslint/no-unused-vars': 'warn', // Downgrade unused variables to warnings
      '@typescript-eslint/consistent-indexed-object-style': 'off', // Allow traditional index signatures
      'no-var': 'off', // Allow var in legacy JS/TS code
      'prefer-const': 'off', // Allow let instead of forcing const
      '@angular-eslint/prefer-inject': 'off', // Allow constructor parameter injection
      '@typescript-eslint/no-empty-function': 'off', // Allow empty methods in tests/mocks
      'no-useless-assignment': 'off', // Allow useless variable assignments
      '@angular-eslint/directive-selector': [
        'error',
        {
          type: 'attribute',
          prefix: 'app',
          style: 'camelCase',
        },
      ],
      '@angular-eslint/component-selector': [
        'error',
        {
          type: 'element',
          prefix: 'app',
          style: 'kebab-case',
        },
      ],
    },
  },
  {
    files: ['**/*.html'],
    extends: [angular.configs.templateRecommended, angular.configs.templateAccessibility],
    rules: {
      '@angular-eslint/template/prefer-control-flow': 'off', // Allow *ngIf and *ngFor instead of forcing @if/@for
      '@angular-eslint/template/label-has-associated-control': 'off', // Allow label elements without explicit controls
      '@angular-eslint/template/click-events-have-key-events': 'off', // Allow click events without key event handlers
      '@angular-eslint/template/mouse-events-have-key-events': 'off', // Allow mouse events without key event handlers
      '@angular-eslint/template/interactive-supports-focus': 'off', // Allow interactive elements without tabindex/focus
      '@angular-eslint/template/role-has-required-aria-props': 'off', // Disable aria role checking
      '@angular-eslint/template/elements-content': 'off', // Allow empty button/icon elements
    },
  },
]);
