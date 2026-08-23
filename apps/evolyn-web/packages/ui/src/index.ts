import type { App } from 'vue';
import {
  EvolynGrid,
  EvolynMemberDepartmentRolePicker,
  EvolynTable,
  VButton,
  VDialog,
} from './components';

export { version } from './version';

const components = [VButton, VDialog, EvolynGrid, EvolynMemberDepartmentRolePicker, EvolynTable];

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
