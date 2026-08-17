import { bench, describe } from 'vitest';
import { mount } from '@vue/test-utils';
import Button from '../components/Button/Button.vue';
import type { ButtonType } from '../components/Button';

describe('Button Performance', () => {
  bench('Button mount', () => {
    mount(Button, {
      slots: {
        default: 'Test Button',
      },
    });
  });

  bench('Button with props mount', () => {
    mount(Button, {
      props: {
        type: 'primary',
        size: 'large',
        disabled: false,
        round: true,
      },
      slots: {
        default: 'Test Button',
      },
    });
  });

  bench('Multiple Button mounts', () => {
    for (let i = 0; i < 100; i++) {
      mount(Button, {
        props: {
          type: i % 2 === 0 ? 'primary' : ('default' as ButtonType),
        },
        slots: {
          default: `Button ${i}`,
        },
      });
    }
  });
});
