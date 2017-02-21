import React, {PropTypes} from 'react';
import { Row, Col, Grid, Table } from 'react-bootstrap';
import { toPairs } from '../utils/utils';
import { connect } from 'react-redux'
import { bindActionCreators } from 'redux'
import actions from '../actions/actions';
import moment from 'moment';

var path = require('path');

class FileTree extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            folderOpen: {},
        };
    }

    handleFolderClick(child, isOpen, e) {
        let folderOpen = this.state.folderOpen;
        let volumeInternalPath = path.join(child.path, child.name);

        folderOpen[volumeInternalPath] = isOpen;
        this.setState({folderOpen});
    }

    humanSize(sz) {
        let K = 1000, M = K*K, G = K*M, T = K*G;

        if (sz < K) {
            return [sz,'B '];
        } else if (sz < M) {
            return [(sz/K).toFixed(2), 'KB'];
        } else if (sz < G) {
            return [(sz/M).toFixed(2), 'MB'];
        } else if (sz < T) {
            return [(sz/G).toFixed(2), 'GB'];
        } else {
            return [(sz/T).toFixed(2), 'TB'];
        }
    }

    downloadVolumeFile(volumeID, file, e) {
        console.log("DOWNLOADING FILE");
        console.log(file);
        console.log(volumeID);
        this.props.actions.downloadVolumeFile(volumeID, file);
        console.log(this.props);
        //("/volumes/{this.props.selectedVolumeID}/"++"?content"
    }

    renderFile(file, indentStyle) {
        let volumeInternalPath = path.join(file.path, file.name);
        let iconClass = "glyphicon-file";
        let browseButton;
        let downloadButton;
        let b2dropButton;
        let isContentVisible = false;
        //console.log(this.props.selectedVolumeID);
        if (file.isFolder) {
            //downloadButton = "";
            //b2dropButton = "";
            iconClass = "glyphicon-folder-close";
            browseButton = <button style={{width:20, background:'none', border:'none', fontSize:20, padding:0}} onClick={ (e) => this.handleFolderClick(file, true, e)}>+</button>
            if (this.state.folderOpen[volumeInternalPath] != null) {
                if (this.state.folderOpen[volumeInternalPath]) {
                    iconClass = "glyphicon-folder-open";
                    browseButton = <button style={{width:20, background:'none', border:'none', fontSize:20, padding:0}} onClick={ (e) => this.handleFolderClick(file, false, e)}>-</button>
                }
            }
        } else {
            downloadButton = <span className="glyphicon glyphicon-download-alt" aria-hidden={true} onClick={ (e) => this.downloadVolumeFile(this.props.selectedVolumeID, file, e)}/>
            b2dropButton = <span className="glyphicon glyphicon-cloud-upload" aria-hidden={true}/>
        }

        if (this.state.folderOpen[file.path] != null) {
            if (this.state.folderOpen[file.path]) {
                isContentVisible = true
            }
        }

        for (var folderName in this.state.folderOpen) {
            if ((file.path.indexOf(folderName) == 0) && (this.state.folderOpen[folderName]==false)) {
                isContentVisible = false;
            }
        }

        if (file.path == "") {
            isContentVisible = true;
        }

        if (isContentVisible) {
            return (
                <li key={volumeInternalPath} className="row file" style={{lineHeight: 2}}>
                    <div className="col-sm-6">
                        <span style={indentStyle}/>
                        {browseButton}
                        <span className={"glyphicon " + iconClass} aria-hidden={true}/> {file.name}
                        {downloadButton}
                        {b2dropButton}
                    </div>
                    <div className="col-sm-3">{this.humanSize(file.size)}</div>
                    <div className="col-sm-3">{moment(file.date).format('ll')}</div>
                </li>
            )
        }
    }

    readVolumeContent(volumeContent, depthLevel, volumeItems) {
        volumeContent.map((fileListItem) => {
            let indentStyle = {paddingLeft: (3*(1+depthLevel))+'em'};

            volumeItems.push(this.renderFile(fileListItem, indentStyle))

            if (fileListItem.isFolder == true) {
                depthLevel += 1;
                volumeItems = this.readVolumeContent(fileListItem.folderTree, depthLevel, volumeItems);
                depthLevel -= 1;
            }
        })
        return volumeItems
    }

    render() {
        //console.log(this.props);
        let sLines = [];
        if (this.props.selectedVolumeContent.length > 0) {
            return (
                <div style={{margin:'1em'}}>
                    <ol className="list-unstyled fileList" style={{textAlign:'left', minHeight:'30em'}}>
                        <li className="heading row" style={{padding:'0.5em 0'}}>
                            <div className="col-sm-6" style={{fontWeight:'bold'}}>File Name</div>
                            <div className="col-sm-3" style={{fontWeight:'bold'}}>Size</div>
                            <div className="col-sm-3" style={{fontWeight:'bold'}}>Date</div>
                        </li>
                        {sLines = this.readVolumeContent(this.props.selectedVolumeContent, 0, [])}
                    </ol>
                </div>
            )
        }
        else {
            return (
                <div>Press Inspect to see files</div>
            )
        }
    }
}

function mapStateToProps(state) {
    return state
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators(actions, dispatch)
    }
}

export default connect(mapStateToProps, mapDispatchToProps)(FileTree);