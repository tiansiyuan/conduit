import React from 'react';
import { Col, Radio, Row } from 'antd';

export default class PageHeader extends React.Component {
  constructor(props) {
    super(props);
    this.onTimeWindowClick = this.onTimeWindowClick.bind(this);
    this.state = {
      selectedWindow: this.props.api.getMetricsWindow()
    };
  }

  onTimeWindowClick(e) {
    let window = e.target.value;
    this.props.api.setMetricsWindow(window);
    this.setState({selectedWindow: window});
  }

  render() {
    return (
      <div className="page-header">
        <Row>
          <Col span={18}>
            {!this.props.header ? null : <h1>{this.props.header}</h1>}
            {!this.props.subHeaderTitle ? null : <div className="subsection-header">{this.props.subHeaderTitle}</div>}
            {!this.props.subHeader ? null : <h1>{this.props.subHeader}</h1>}
          </Col>
          <Col span={6}>
            <Radio.Group
              className="time-window-btns"
              value={this.state.selectedWindow}
              onChange={this.onTimeWindowClick} >
              <Radio.Button value="10s">10s</Radio.Button>
              <Radio.Button value="1m">1m</Radio.Button>
              <Radio.Button value="10m">10m</Radio.Button>
              <Radio.Button value="1h">1h</Radio.Button>
            </Radio.Group>
          </Col>
        </Row>

        {!this.props.subMessage ? null : <div>{this.props.subMessage}</div>}
      </div>
    );
  }
}
