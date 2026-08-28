import { withInstall } from '~/_utils';
import EvolynIconPickerComponent from './EvolynIconPicker.vue';

export const EvolynIconPicker = withInstall(EvolynIconPickerComponent);
export default EvolynIconPicker;

export * from './EvolynIconPicker.types';
export { defaultIconColors, defaultSystemIcons } from './iconOptions';
