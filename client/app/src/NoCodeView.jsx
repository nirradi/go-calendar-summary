import React from 'react';
import { Button } from 'antd';
import axios from 'axios'

class NoCodeView extends React.Component {

  constructor(props) {
    super(props);
    this.state = {link: ""};
    axios.get('http://127.0.0.1:37555?format=json')
      .then( (response)  => {
        // handle success
        this.setState({link: response.data.auth})
      })
  }

  render() {
    return (
        <div>
            <Button disabled={!this.state.link} href={this.state.link} type="primary">Allow calendar access</Button>
        </div>
    );
  }
}

export default NoCodeView;
