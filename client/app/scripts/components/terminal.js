const React = require('react');

const Term = require('../vendor/term.js');

const wsProto = location.protocol === 'https:' ? 'wss' : 'ws';
const wsUrl = __WS_URL__ || wsProto + '://' + location.host + location.pathname.replace(/\/$/, '');

function ab2str(buf) {
  return String.fromCharCode.apply(null, new Uint8Array(buf));
}

const Terminal = React.createClass({

  componentDidMount: function() {
    let socket = new WebSocket(wsUrl + '/api/pipe/' + this.props.controlPipe);
    socket.binaryType = 'arraybuffer';
    const component = this;

    const term = new Term({
      cols: 80,
      rows: 24,
      screenKeys: true
    });

    term.on('data', function(data) {
      socket.send(data);
    });

    socket.onopen = function() {
      term.open(component.inner.getDOMNode());
      term.write('\x1b[31mWelcome to term.js!\x1b[m\r\n');
    };

    socket.onclose = function() {
      console.log('socket closed');
      term.destroy();
      socket = null;
    };

    socket.onerror = function() {
      console.error('socket error');
    };

    socket.onmessage = function(event) {
      console.log('pipe data', event.data.size);
      term.write(ab2str(event.data));
    };
  },

  render: function() {
    return (
      <div id="terminal">
        <div style={{height: '100%', paddingBottom: 8, borderRadius: 2,
          backgroundColor: '#fff',
          boxShadow: '0 10px 30px rgba(0, 0, 0, 0.19), 0 6px 10px rgba(0, 0, 0, 0.23)',
          fontFamily: '"DejaVu Sans Mono", "Liberation Mono", monospace',
          color: '#f0f0f0'
        }}
          ref={(ref) => this.inner = ref}>
          Pipe: {this.props.controlPipe}
        </div>
      </div>
    );
  }

});

module.exports = Terminal;
