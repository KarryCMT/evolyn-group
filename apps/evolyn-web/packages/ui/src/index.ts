import type { App } from 'vue';
import {
  EvolynButton,
  EvolynChart,
  EvolynDialog,
  EvolynGrid,
  EvolynIconPicker,
  EvolynMemberDepartmentRolePicker,
  EvolynRichTextEditor,
  EvolynScrollbar,
  EvolynTable,
} from './components';

export { version } from './version';

const components = [
  EvolynButton,
  EvolynChart,
  EvolynDialog,
  EvolynGrid,
  EvolynIconPicker,
  EvolynMemberDepartmentRolePicker,
  EvolynRichTextEditor,
  EvolynScrollbar,
  EvolynTable,
];

function install(app: App) {
  components.forEach((component) => {
    app.use(component);
  });
}

export { install };

export * from './components';

export default {
  install,
};
