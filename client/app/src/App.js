import React from 'react';
import { Layout } from 'antd';
import CalendarList from './CalenderList'
import NoCodeView from './NoCodeView'

const { Header, Content, Footer } = Layout;

class App extends React.Component {


  render() {
    console.log(this.props.code)
    var content = this.props.code ? <CalendarList code={this.props.code}/> : <NoCodeView />
    return (
      <Layout style={{ minHeight: '100vh' }}>
        <Layout>
          <Header style={{ background: '#fff', padding: 12 }} ><h1>Meeting Summary</h1></Header>
          <Content style={{ margin: '0 16px' }}>
            {content}
          </Content>
          <Footer style={{ textAlign: 'center' }}>Created by NiRR 2019</Footer>
        </Layout>
      </Layout>
    );
  }
}

export default App;
