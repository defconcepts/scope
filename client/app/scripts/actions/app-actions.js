let ActionTypes;
let AppDispatcher;
let AppStore;
let RouterUtils;
let WebapiUtils;

module.exports = {
  changeTopologyOption: function(option, value, topologyId) {
    AppDispatcher.dispatch({
      type: ActionTypes.CHANGE_TOPOLOGY_OPTION,
      topologyId: topologyId,
      option: option,
      value: value
    });
    RouterUtils.updateRoute();
    // update all request workers with new options
    WebapiUtils.getTopologies(
      AppStore.getActiveTopologyOptions()
    );
    WebapiUtils.getNodesDelta(
      AppStore.getCurrentTopologyUrl(),
      AppStore.getActiveTopologyOptions()
    );
    WebapiUtils.getNodeDetails(
      AppStore.getCurrentTopologyUrl(),
      AppStore.getSelectedNodeId(),
      AppStore.getActiveTopologyOptions()
    );
  },

  clickCloseDetails: function() {
    AppDispatcher.dispatch({
      type: ActionTypes.CLICK_CLOSE_DETAILS
    });
    RouterUtils.updateRoute();
  },

  clickNode: function(nodeId) {
    AppDispatcher.dispatch({
      type: ActionTypes.CLICK_NODE,
      nodeId: nodeId
    });
    RouterUtils.updateRoute();
    WebapiUtils.getNodeDetails(
      AppStore.getCurrentTopologyUrl(),
      AppStore.getSelectedNodeId(),
      AppStore.getActiveTopologyOptions()
    );
  },

  clickTopology: function(topologyId) {
    AppDispatcher.dispatch({
      type: ActionTypes.CLICK_TOPOLOGY,
      topologyId: topologyId
    });
    RouterUtils.updateRoute();
    WebapiUtils.getNodesDelta(
      AppStore.getCurrentTopologyUrl(),
      AppStore.getActiveTopologyOptions()
    );
  },

  openWebsocket: function() {
    AppDispatcher.dispatch({
      type: ActionTypes.OPEN_WEBSOCKET
    });
  },

  clearControlError: function() {
    AppDispatcher.dispatch({
      type: ActionTypes.CLEAR_CONTROL_ERROR
    });
  },

  closeWebsocket: function() {
    AppDispatcher.dispatch({
      type: ActionTypes.CLOSE_WEBSOCKET
    });
  },

  doControl: function(probeId, nodeId, control) {
    AppDispatcher.dispatch({
      type: ActionTypes.DO_CONTROL
    });
    WebapiUtils.doControl(
      probeId,
      nodeId,
      control
    );
  },

  enterEdge: function(edgeId) {
    AppDispatcher.dispatch({
      type: ActionTypes.ENTER_EDGE,
      edgeId: edgeId
    });
  },

  enterNode: function(nodeId) {
    AppDispatcher.dispatch({
      type: ActionTypes.ENTER_NODE,
      nodeId: nodeId
    });
  },

  hitEsc: function() {
    AppDispatcher.dispatch({
      type: ActionTypes.HIT_ESC_KEY
    });
    RouterUtils.updateRoute();
  },

  leaveEdge: function(edgeId) {
    AppDispatcher.dispatch({
      type: ActionTypes.LEAVE_EDGE,
      edgeId: edgeId
    });
  },

  leaveNode: function(nodeId) {
    AppDispatcher.dispatch({
      type: ActionTypes.LEAVE_NODE,
      nodeId: nodeId
    });
  },

  receiveControlError: function(err) {
    AppDispatcher.dispatch({
      type: ActionTypes.DO_CONTROL_ERROR,
      error: err
    });
  },

  receiveControlPipe: function(pipeId) {
    AppDispatcher.dispatch({
      type: ActionTypes.RECEIVE_CONTROL_PIPE,
      pipeId: pipeId
    });
  },

  receiveControlSuccess: function() {
    AppDispatcher.dispatch({
      type: ActionTypes.DO_CONTROL_SUCCESS
    });
  },

  receiveNodeDetails: function(details) {
    AppDispatcher.dispatch({
      type: ActionTypes.RECEIVE_NODE_DETAILS,
      details: details
    });
  },

  receiveNodesDelta: function(delta) {
    AppDispatcher.dispatch({
      type: ActionTypes.RECEIVE_NODES_DELTA,
      delta: delta
    });
  },

  receiveTopologies: function(topologies) {
    AppDispatcher.dispatch({
      type: ActionTypes.RECEIVE_TOPOLOGIES,
      topologies: topologies
    });
    WebapiUtils.getNodesDelta(
      AppStore.getCurrentTopologyUrl(),
      AppStore.getActiveTopologyOptions()
    );
    WebapiUtils.getNodeDetails(
      AppStore.getCurrentTopologyUrl(),
      AppStore.getSelectedNodeId(),
      AppStore.getActiveTopologyOptions()
    );
  },

  receiveApiDetails: function(apiDetails) {
    AppDispatcher.dispatch({
      type: ActionTypes.RECEIVE_API_DETAILS,
      version: apiDetails.version
    });
  },

  receiveError: function(errorUrl) {
    AppDispatcher.dispatch({
      errorUrl: errorUrl,
      type: ActionTypes.RECEIVE_ERROR
    });
  },

  route: function(state) {
    AppDispatcher.dispatch({
      state: state,
      type: ActionTypes.ROUTE_TOPOLOGY
    });
    WebapiUtils.getTopologies(
      AppStore.getActiveTopologyOptions()
    );
    WebapiUtils.getNodesDelta(
      AppStore.getCurrentTopologyUrl(),
      AppStore.getActiveTopologyOptions()
    );
    WebapiUtils.getNodeDetails(
      AppStore.getCurrentTopologyUrl(),
      AppStore.getSelectedNodeId(),
      AppStore.getActiveTopologyOptions()
    );
  }
};

// require below export to break circular dep

AppDispatcher = require('../dispatcher/app-dispatcher');
ActionTypes = require('../constants/action-types');

RouterUtils = require('../utils/router-utils');
WebapiUtils = require('../utils/web-api-utils');
AppStore = require('../stores/app-store');
