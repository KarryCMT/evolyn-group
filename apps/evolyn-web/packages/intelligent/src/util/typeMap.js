export const valueCollectorMap = {
  option: {
    label: '默认选项',
    value: 'option',
    icon: 'el-icon-folder-opened'
  },
  input: {
    label: '手动输入',
    value: 'input',
    icon: 'el-icon-edit-outline'
  },
  component: {
    label: '页面组件值',
    value: 'component',
    icon: 'el-icon-cpu'
  },
  componentProp: {
    label: '页面组件属性值',
    value: 'componentProp',
    icon: 'el-icon-connection'
  },
  dataSource: {
    label: '数据节点返回值',
    value: 'dataSource',
    icon: 'el-icon-receiving'
  },
  dataConvert: {
    label: '转换节点返回值',
    value: 'dataConvert',
    icon: 'el-icon-s-operation'
  },
  urlParam: {
    label: '路由参数属性值',
    value: 'urlParam',
    icon: 'el-icon-link'
  },
  initParam: {
    label: '宿主系统变量属性值',
    value: 'initParam',
    icon: 'el-icon-attract'
  }
}

export const commonNodeMap = {
  dataSource: {
    label: '请求数据',
    value: 'dataSource',
    logo: ''
  },
  pageJump: {
    label: '页面跳转',
    value: 'pageJump',
    logo: ''
  },
  dataConvert: {
    label: '数据转换',
    value: 'dataConvert',
    logo: ''
  },
}

export const eventNodeMap = {
  pageInit: {
    label: '页面初始化',
    value: 'pageInit',
    logo: ''
  }
}

export const toolMap = {
  undo: {
    name: 'undo',
    desc: '返回上一步',
    icon: ''
  },
  redo: {
    name: 'redo',
    desc: '恢复下一步',
    icon: ''
  },
  zoomIn: {
    name: 'zoomIn',
    desc: '画布放大',
    icon: ''
  },
  zoomOut: {
    name: 'zoomOut',
    desc: '画布缩小',
    icon: ''
  },
  fitView: {
    name: 'fitView',
    desc: '区域适应',
    icon: ''
  },
  selection: {
    name: 'selection',
    desc: '框选节点',
    icon: ''
  },
  beautify: {
    name: 'beautify',
    desc: '美化布局',
    icon: ''
  },
  navigation: {
    name: 'navigation',
    desc: '全图导航',
    icon: ''
  }
}

export const defaultLogo = ''

export const requestMethodMap = [
  {
    value: 'GET',
    label: 'GET'
  },
  {
    value: 'POST',
    label: 'POST'
  }
]
