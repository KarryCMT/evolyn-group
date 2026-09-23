import apps from './app';
import auths from './auth';
import dashboards from './dashboard';
import forms from './form';
import qr from './qr';
import tenants from './tenant';
import workflows from './workflow';

export default [...auths, ...qr, ...dashboards, ...tenants, ...apps, ...forms, ...workflows];
